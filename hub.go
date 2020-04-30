// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wsPool

import (
	"container/list"
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
	list [][]byte
	Expiration time.Time //过期时间
}



func newHub() *hub {
	return &hub{
		broadcast:  	make(chan *SendMsg,64),
		chanBroadcast:  make(chan *SendMsg,64),
		register:   	make(chan *Client,1024),
		unregister: 	make(chan string,1024),
		clients:    	gmap.NewStrAnyMap(true),//make(map[string]*Client),//
		broadcastQueue:queue.NewPriorityQueue(),
		chanBroadcastQueue:queue.NewPriorityQueue(),
	}
}

func (h *hub) run() {
	ticker := make(chan int)
	gtimer.AddSingleton(10*time.Second, func() {
		ticker <- 1
	})
	for {
		select {
		case id := <-h.unregister:
			log.Println("取消注册ws连接对象：", id, "连接总数：", h.clients.Size())
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
					if client.sendChQueue.Len() == 0 {
						close(client.sendCh)
						close(client.ping)
						client.sendCh=nil
						client.ping=nil
						h.clients.Remove(id)
					} else {
						client.CloseTime = time.Now()
					}
				}
			}

		case client := <-h.register:
			log.Println("注册ws连接对象：", client.Id, "连接总数：", h.clients.Size())
			cl := h.clients.Get(client.Id)
			if cl != nil {
				c := cl.(*Client)
				//恢复之前连接里面存的在未处理的队列
				if c.sendChQueue.Len() > 0 {
					var next *list.Element
					pq := c.sendChQueue
					for iter := pq.Data.Front(); iter != nil; iter = next {
						//fmt.Println("item:", iter.Value.(*Item))
						next = iter.Next()
						v := iter.Value.(*queue.Item)
						client.sendChQueue.Push(v)
					}
				}
				c.sendChQueue = nil
				close(c.sendCh)
				close(c.ping)
				c.sendCh=nil
				c.ping=nil
				h.clients.Remove(c.Id)
			}
			h.clients.Set(client.Id, client)
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

		case <-ticker: //断线超过一分钟清除连接消息队列缓存数据
			list:=h.clients.Values()
			for _,client:=range list {
				if client != nil {
					v := client.(*Client)
					if v != nil && v.IsClose {
						isExpri := v.CloseTime.Add(60 * time.Second).Before(time.Now())
						if isExpri {
							v.sendChQueue = nil
							if v.sendCh!=nil {
								close(v.sendCh)
							}
							if v.ping!=nil {
								close(v.ping)
							}
							h.clients.Remove(v.Id)
						}
						log.Println("清理ws连接对象，连接总数：", h.clients.Size())
					}
				}
			}
		}
	}
}

func (h *hub) ticker() {
	gtimer.AddSingleton(5*time.Millisecond, func() {
		n := len(h.broadcast)
		if(n==0&&h.broadcastQueue.Len()>0) {
			item:=h.broadcastQueue.Pop()
			if item!=nil {
				msg:=item.Data.(*SendMsg)
				broadcastAll(msg)
			}
		}
		n = len(h.chanBroadcast)
		if(n==0&&h.chanBroadcastQueue.Len()>0) {
			item:=h.chanBroadcastQueue.Pop()
			if item!=nil {
				msg:=item.Data.(*SendMsg)
				broadcast(msg)
			}
		}
	})

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

