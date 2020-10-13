// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wsPool

import (
	"errors"
	"gitee.com/rczweb/wsPool/util/gmap"
	"log"
	"time"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type hub struct {
	// Registered clients.
	clients *gmap.StrAnyMap  //map[string]*Client// //新的处理方式有没有线程效率会更高,所以SafeMap的锁处理都去掉了
	//oldMsgQueue *gmap.StrAnyMap  //缓存断开的连接消息队列

	// Inbound messages from the clients.
	//可以用于广播所有连接对象
	broadcast chan broadcastMessage

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan string
}

type broadcastMessage struct {
	Channel string
	Msg sendMessage
}


func newHub() *hub {
	return &hub{
		register:   	make(chan *Client,2048),
		unregister: 	make(chan string,2048),
		clients:    	gmap.NewStrAnyMap(true),//make(map[string]*Client),//
		broadcast: make(chan broadcastMessage,2048),
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
		case message, ok := <-h.broadcast:
			if ok {
				if message.Channel=="0" { //全局广播
					h.clients.Iterator(func(id string, v interface{}) bool {
						if v != nil {
							client := v.(*Client)
							if !client.IsClose{
								client.send(message.Msg)
							}
						}
						return true
					})
				}else{
					h.clients.Iterator(func(id string, v interface{}) bool {
						if v != nil {
							client := v.(*Client)
							if !client.IsClose &&searchStrArray(client.channel, message.Channel){
								client.send(message.Msg)
							}
						}
						return true
					})
				}
				//broadcastAll(message)
			}
		}
	}
}


func (h *hub) brostcastMsg(message broadcastMessage) error {
	timeout := time.NewTimer(time.Millisecond * 800)
	select {
	case h.broadcast<-message:
		return nil
	case <-timeout.C:
		return errors.New("chanBroadcastQueue消息管道blocked,写入消息超时")
	}
}

