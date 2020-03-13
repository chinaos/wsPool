// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wsPool

import (
	"time"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	//可以用于广播所有连接对象
	broadcast chan []byte

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
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		reconectSendMsg: make(map[*oldMsg]string),
	}
}

func (h *hub) run() {
	ticker := time.NewTicker(1*time.Second)
	defer dump()
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
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
							client.send<-v
						}
					}
				}

			}

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				//此处需要把原来channel中没处理完的消息添加回队列中
				n := len(client.send)
				if n>0 {
					list:=make([][]byte,0)
					for i := 0; i < n; i++ {
						msg,_:=<-client.send
						list=append(list,msg)
					}
					h.reconectSendMsg[&oldMsg{
						list:list,
						Expiration:time.Now().Add(60 * time.Second),
					}]=client.Id
				}
				close(client.send)
			}

		case message := <-h.broadcast:
			//全局广播消息处理
			for client := range h.clients {
				select {
				case client.send <- message:
					/*default:
						close(client.send)
						delete(h.clients, client)*/
				}
			}

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
