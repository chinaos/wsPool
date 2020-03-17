// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wsPool

import (
	"errors"
	"time"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type hub struct {
	// Registered clients.
	clients  *SafeMap  //新的处理方式有没有线程效率会更高,所以SafeMap的锁处理都去掉了

	// Inbound messages from the clients.
	//可以用于广播所有连接对象
	broadcast chan []byte

	//广播指定频道的管道
	chanBroadcast chan *SendMsg

	//全局指定发消息通过ToClientId发送消息
	sendByToClientId chan *SendMsg

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	//重新连接需要处理的消息(缓存上次未来得能处理发送消息channel中的消息，20秒后原ws未连接消息失效)
	reconectSendMsg map[*oldMsg]string

}

//重新连接需要处理的消息(缓存上次未来得能处理发送消息channel中的消息，60秒后原ws未连接消息失效)
type oldMsg struct {
	list [][]byte
	Expiration time.Time //过期时间
}



func newHub() *hub {
	return &hub{
		broadcast:  	make(chan []byte,1024),
		chanBroadcast:  make(chan *SendMsg,2048),
		sendByToClientId:  make(chan *SendMsg,2048),
		register:   	make(chan *Client),
		unregister: 	make(chan *Client),
		clients:    	NewSafeMap(),
		reconectSendMsg: make(map[*oldMsg]string),
	}
}

func (h *hub) run() {
	ticker := time.NewTicker(1*time.Second)
	defer dump()
	for {
		select {
		case client := <-h.register:
			h.clients.Set(client,true)
			for oldmsg,clientid:=range h.reconectSendMsg{
				if oldmsg==nil {
					continue
				}
				if clientid==""{
					delete(h.reconectSendMsg,oldmsg)
					continue
				}
				if clientid==client.Id {
					if len(oldmsg.list)>0 {
						//有消息添加的channel中
						for _,v:=range oldmsg.list{
							client.sendCh<-v
						}
					}
				}

			}

		case client := <-h.unregister:
			if h.clients.Check(client){
				h.clients.Delete(client)
				//此处需要把原来channel中没处理完的消息添加回队列中
				n := len(client.sendCh)
				if n>0 {
					list:=make([][]byte,0)
					for i := 0; i < n; i++ {
						msg,_:=<-client.sendCh
						list=append(list,msg)
					}
					h.reconectSendMsg[&oldMsg{
						list:list,
						Expiration:time.Now().Add(60 * time.Second),
					}]=client.Id
				}
				close(client.sendCh)
				close(client.ping)
			}

		case message := <-h.broadcast:
			//全局广播消息处理
			h.clients.Iterator(func(k *Client, v bool) bool {
				k.send(message)
				return true
			})


		case message := <-h.chanBroadcast:
			//广播指定频道的消息处理
			wsSever.hub.clients.Iterator(func(client *Client,v bool ) bool {
				//找到ToClientId对应的连接对象
				if client.IsClose {
					return true
				}
				for _,ch:=range message.Channel  {
					if searchStrArray(client.channel,ch){
						message.ToClientId=client.Id
						err := client.Send(message)
						if err != nil {
							client.onError(errors.New("连接ID："+client.Id+"广播消息出现错误："+ err.Error()))
						}
					}
				}
				return true
			})



		case message := <-h.sendByToClientId:

			//包级发送消息指定sendByToClientId
			wsSever.hub.clients.Iterator(func(client *Client, v bool) bool {
				//找到ToClientId对应的连接对象
				if client.Id==message.ToClientId{
					if client.IsClose {
						client.onError( errors.New("包级发送消息sendByToClientId错误：连接对像："+client.Id+"连接状态异常，连接己经关闭！"))
						return false
					}
					message.ToClientId=client.Id
					err := client.Send(message)
					if err != nil {
						client.onError(errors.New("发送消息出错："+ err.Error()+",连接对象id="+client.Id+"。"))
						return false
					}
					return false
				}
				return true
			})


		case <-ticker.C:
			//定时清理连接断开后未处理的消息
			for oldmsg,clientid:=range h.reconectSendMsg{
				if oldmsg==nil {
					continue
				}
				if clientid==""{
					delete(h.reconectSendMsg,oldmsg)
					continue
				}
				isExpri:=oldmsg.Expiration.Before(time.Now())
				if isExpri {
					delete(h.reconectSendMsg,oldmsg)
				}
			}

		}
	}
}
