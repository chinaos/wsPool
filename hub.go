// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wsPool

import (
	"container/list"
	"errors"
	"gitee.com/rczweb/wsPool/queue"
	"log"
	"time"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type hub struct {
	// Registered clients.
	clients map[string]*Client// *SafeMap  //新的处理方式有没有线程效率会更高,所以SafeMap的锁处理都去掉了

	// Inbound messages from the clients.
	//可以用于广播所有连接对象
	broadcast chan *SendMsg
	broadcastQueue *queue.PriorityQueue

	//广播指定频道的管道
	chanBroadcast chan *SendMsg
	chanBroadcastQueue *queue.PriorityQueue

	//全局指定发消息通过ToClientId发送消息
	sendByToClientId chan *SendMsg
	sendByToClientIdQueue *queue.PriorityQueue

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
		broadcast:  	make(chan *SendMsg,256),
		chanBroadcast:  make(chan *SendMsg,256),
		sendByToClientId:  make(chan *SendMsg,256),
		register:   	make(chan *Client,1024),
		unregister: 	make(chan string),
		clients:    	make(map[string]*Client),//NewSafeMap(),
		broadcastQueue:queue.NewPriorityQueue(),
		chanBroadcastQueue:queue.NewPriorityQueue(),
		sendByToClientIdQueue:queue.NewPriorityQueue(),
	}
}

func (h *hub) run() {
	ticker := time.NewTicker(60*time.Second)
	for {
		select {
		case id := <-h.unregister:
			log.Println("取消注册ws连接对象：",id,"连接总数：",len(h.clients))
			client:=h.clients[id]
			//这里有可能连接得新连接进来后还没有关掉，这时此对象会被重置为新的连接对象
			if client!=nil&&client.IsClose{
				n:=len(client.sendCh)
				if n>0{
					for i:=0;i<n;i++ {
						client.sendChQueue.Push(&queue.Item{
							Data:<-client.sendCh,
							Priority:1,
						})
					}
				}
				if client.sendChQueue.Len()==0 {
					close(client.sendCh)
					close(client.ping)
					h.clients[id]=nil
					delete(h.clients,id)
				}else{
					client.CloseTime=time.Now()
				}
			}
		case client := <-h.register:
			log.Println("注册ws连接对象：",client.Id,"连接总数：",len(h.clients))
			c:=h.clients[client.Id]
			if c!=nil {
				//恢复之前连接里面存的在未处理的队列
				if c.sendChQueue.Len()>0 {
					var next *list.Element
					pq:=c.sendChQueue
					for iter := pq.Data.Front(); iter!=nil;iter=next {
						//fmt.Println("item:", iter.Value.(*Item))
						v := iter.Value.(*queue.Item)
						client.sendChQueue.Push(v)
						next=iter.Next()
					}
				}
				c.sendChQueue=nil
				close(c.sendCh)
				close(c.ping)
				h.clients[c.Id]=nil
				delete(h.clients,c.Id)
			}
			h.clients[client.Id]=client
		case message := <-h.broadcast:
			//log.Println("全局广播消息：",message,"消息总数：",len(h.broadcast))
			//全局广播消息处理
			for _,v:=range h.clients{
				if v!=nil&&!v.IsClose {
					 v.Send(message)
				}

			}
			n := len(h.broadcast)
			if(n>0) {
				for i := 0; i < n; i++ {
					msg := <-h.broadcast
					for _, v := range h.clients {
						if v != nil && !v.IsClose {
							v.Send(msg)
						}
					}

				}
			}
		case message := <-h.chanBroadcast:
			//log.Println("频道广播消息：",message,"消息总数：",len(h.chanBroadcast))
			//广播指定频道的消息处理
			for id,client:=range h.clients{
				if client!=nil {
					if client.IsClose {
						continue
					}
					for _,ch:=range message.Channel  {
						if searchStrArray(client.channel,ch){
							message.ToClientId=id
							err:=client.Send(message)
							if err != nil {
								client.onError(errors.New("连接ID："+client.Id+"广播消息出现错误："+ err.Error()))
							}
						}
					}
				}

			}
			n := len(h.chanBroadcast)
			if(n>0) {
				for i := 0; i < n; i++ {
					msg := <-h.chanBroadcast
					for id, client := range h.clients {
						if client != nil {
							if client.IsClose {
								continue
							}
							for _, ch := range msg.Channel {
								if searchStrArray(client.channel, ch) {
									msg.ToClientId = id
									err := client.Send(msg)
									if err != nil {
										client.onError(errors.New("连接ID：" + client.Id + "广播消息出现错误：" + err.Error()))
									}
								}
							}
						}

					}

				}
			}
		case message := <-h.sendByToClientId:
			//log.Println("包级发送消息指定sendByToClientId：",message,"消息总数：",len(h.sendByToClientId))
			//包级发送消息指定sendByToClientId

			for id,client:=range h.clients{
				if client!=nil {
					if client.Id==message.ToClientId{
						if client.IsClose {
							client.onError( errors.New("包级发送消息sendByToClientId错误：连接对像："+client.Id+"连接状态异常，连接己经关闭！"))
							break
						}
						message.ToClientId=id
						err := client.Send(message)
						if err != nil {
							client.onError(errors.New("发送消息出错："+ err.Error()+",连接对象id="+client.Id+"。"))
						}
						break
					}
				}

			}
			n := len(h.sendByToClientId)
			if(n>0) {
				for i := 0; i < n; i++ {
					msg := <-h.sendByToClientId
					for id, client := range h.clients {
						if client != nil {
							if client.Id == msg.ToClientId {
								/*if client.IsClose {
									client.onError(errors.New("包级发送消息sendByToClientId错误：连接对像：" + client.Id + "连接状态异常，连接己经关闭！"))
									break
								}*/
								msg.ToClientId = id
								err := client.Send(msg)
								if err != nil {
									client.onError(errors.New("发送消息出错：" + err.Error() + ",连接对象id=" + client.Id + "。"))
								}
								break
							}
						}

					}
				}
			}
		case <-ticker.C://断线超过一分钟清除所有队列缓存数据
			for _,v:=range h.clients{
				if v!=nil&&v.IsClose {
					isExpri:=v.CloseTime.Add(60*time.Second).Before(time.Now())
					if isExpri {
						v.sendChQueue=nil
						close(v.sendCh)
						close(v.ping)
						h.clients[v.Id]=nil
						delete(h.clients,v.Id)
					}
					}
				}
			log.Println("清理ws连接对象，连接总数：",len(h.clients))
		}
	}
}

func (h *hub) ticker() {
	tk := time.NewTicker(10*time.Millisecond)
	for{
		select {
			case <-tk.C:
			n := len(h.broadcast)
			if(n<255&&h.broadcastQueue.Len()>0) {
				item:=h.broadcastQueue.Pop()
				if item!=nil {
					msg:=item.Data.(*SendMsg)
					broadcastAll(msg)
				}
			}
			n = len(h.chanBroadcast)
			if(n<255&&h.chanBroadcastQueue.Len()>0) {
				item:=h.chanBroadcastQueue.Pop()
				if item!=nil {
					msg:=item.Data.(*SendMsg)
					broadcast(msg)
				}
			}

			n = len(h.sendByToClientId)
			if(n<255&&h.sendByToClientIdQueue.Len()>0) {
				item:=h.sendByToClientIdQueue.Pop()
				if item!=nil {
					msg:=item.Data.(*SendMsg)
					send(msg)
				}
			}
		}
	}
}

