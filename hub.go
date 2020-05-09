// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wsPool

import (
	"errors"
	"gitee.com/rczweb/wsPool/util/queue"
	"github.com/gogf/gf/container/gmap"
	"github.com/gogf/gf/os/gtimer"
	"log"
	"time"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type hub struct {
	// Registered clients.
	clients *gmap.StrAnyMap  //map[string]*Client// //新的处理方式有没有线程效率会更高,所以SafeMap的锁处理都去掉了
	oldMsgQueue *gmap.StrAnyMap  //缓存断开的连接消息队列

	// Inbound messages from the clients.
	//可以用于广播所有连接对象
	broadcast chan *SendMsg
	broadcastQueue *queue.PriorityQueue

	//广播指定频道的管道
	chanBroadcast chan *SendMsg
	chanBroadcastQueue *queue.PriorityQueue

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan string
}

//重新连接需要处理的消息(缓存上次未来得能处理发送消息channel中的消息，60秒后原ws未连接消息失效)
type oldMsg struct {
	list *queue.PriorityQueue
	Expiration time.Time //过期时间
}



func newHub() *hub {
	return &hub{
		broadcast:  	make(chan *SendMsg,64),
		chanBroadcast:  make(chan *SendMsg,64),
		register:   	make(chan *Client,20480),
		unregister: 	make(chan string,20480),
		clients:    	gmap.NewStrAnyMap(true),//make(map[string]*Client),//
		oldMsgQueue:    	gmap.NewStrAnyMap(true),//缓存断开的连接消息队列
		broadcastQueue:queue.NewPriorityQueue(),
		chanBroadcastQueue:queue.NewPriorityQueue(),
	}
}

func (h *hub) run() {
	for {
		select {
		case id := <-h.unregister:
			c := h.clients.Get(id)
			if c != nil {
				client := c.(*Client)
				//这里有可能连接得新连接进来后还没有关掉，这时此对象会被重置为新的连接对象
				if client != nil && client.IsClose {
					n := len(client.sendCh)
					if n > 0 {
						for i := 0; i < n; i++ {
							client.sendChQueue.Push(&queue.Item{
								Data:       <-client.sendCh,
								Priority:   1,
								AddTime:    time.Now(),
								Expiration: 60,
							})
						}
					}
					h.oldMsgQueue.Set(id, &oldMsg{
						list:client.sendChQueue,
						Expiration:time.Now().Add(time.Duration(120)*time.Second),
					})
					close(client.sendCh)
					close(client.ping)
					client.sendCh = nil
					client.ping = nil
					h.clients.Remove(id)
				}
			}
			log.Println("取消注册ws连接对象：", id, "连接总数：", h.clients.Size())

		case client := <-h.register:
			log.Println("注册ws连接对象：", client.Id, "连接总数：", h.clients.Size())
			client.CloseTime = -1
			h.clients.Set(client.Id, client)

		}
	}
}
/*
func (h *hub) runbroadcast() {
	for {
		select {

		case message := <-h.broadcast:
			//log.Println("全局广播消息：",message,"消息总数：",len(h.broadcast))
			//全局广播消息处理
			h.clients.Iterator(func(id string, v interface{}) bool {
				if v != nil {
					client := v.(*Client)
					if !client.IsClose {
						message.ToClientId = id
						client.Send(message)
					}
				}
				return true
			})

		case message := <-h.chanBroadcast:
			//log.Println("频道广播消息：",message,"消息总数：",len(h.chanBroadcast))
			//广播指定频道的消息处理
			h.clients.Iterator(func(id string, v interface{}) bool {
				if v != nil {
					client := v.(*Client)
					if !client.IsClose {
						for _, ch := range message.Channel {
							if searchStrArray(client.channel, ch) {
								message.ToClientId = id
								err := client.Send(message)
								if err != nil {
									client.onError(errors.New("连接ID：" + client.Id + "广播消息出现错误：" + err.Error()))
								}
							}
						}
					}
				}
				return true
			})
		}
	}
}*/

func (h *hub) ticker() {
	//定时清理连接对象
	gtimer.AddSingleton(10*time.Second, func() {
		list:=h.oldMsgQueue.Keys()
		for _,key:=range list {
			oldmsg:=h.oldMsgQueue.Get(key)
			if oldmsg != nil {
				v := oldmsg.(*oldMsg)
				if v != nil {
					isExpri := v.Expiration.Before(time.Now())
					if isExpri {
						h.oldMsgQueue.Remove(key)
					}
					log.Println("清理ws连接池对象缓存的旧消息：", h.oldMsgQueue.Size())
				}
			}
		}
	})
	//定时发送广播队列
	gtimer.AddSingleton(500*time.Microsecond, func() {
		n := len(h.broadcast)
		if(n==0&&h.broadcastQueue.Len()>0) {
			item:=h.broadcastQueue.Pop()
			if item!=nil {
				message:=item.Data.(*SendMsg)
				//全局广播消息处理
				h.clients.Iterator(func(id string, v interface{}) bool {
					if v != nil {
						client := v.(*Client)
						if !client.IsClose {
							message.ToClientId = id
							client.Send(message)
						}
					}
					return true
				})
				//broadcastAll(message)
			}
		}
		n = len(h.chanBroadcast)
		if(n==0&&h.chanBroadcastQueue.Len()>0) {
			item:=h.chanBroadcastQueue.Pop()
			if item!=nil {
				message:=item.Data.(*SendMsg)

				//广播指定频道的消息处理
				h.clients.Iterator(func(id string, v interface{}) bool {
					if v != nil {
						client := v.(*Client)
						if !client.IsClose {
							for _, ch := range message.Channel {
								if searchStrArray(client.channel, ch) {
									message.ToClientId = id
									err := client.Send(message)
									if err != nil {
										client.onError(errors.New("连接ID：" + client.Id + "广播消息出现错误：" + err.Error()))
									}
								}
							}
						}
					}
					return true
				})
				//broadcast(message)
			}
		}
	})
	//定时清理广播队列超时的消息
	gtimer.AddSingleton(30*time.Second, func() {
 		//定时检查队列中过期的消息
		h.clearExpirationsBroadcastMessage()
	})

}

func (h *hub)clearExpirationsBroadcastMessage()  {
	h.broadcastQueue.Expirations(func(item *queue.Item) {
		message:=item.Data.(*SendMsg)
		c:=h.clients.Get(message.ToClientId)
		if c!=nil {
			client:=c.(*Client)
			if client.Id==message.ToClientId&& !client.IsClose{
				client.grpool.Add(func() {
					client.expirationsBroadcastMessage(message)
				})
			}
		}

	})

	h.chanBroadcastQueue.Expirations(func(item *queue.Item) {
		message:=item.Data.(*SendMsg)
		c:=h.clients.Get(message.ToClientId)
		if c!=nil {
			client:=c.(*Client)
			if client.Id == message.ToClientId && !client.IsClose {
				client.grpool.Add(func() {
					client.expirationsBroadcastMessage(message)
				})
			}
		}
	})
}

