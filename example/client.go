// Copyright 2015 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package main

import (
	"flag"
	"gitee.com/rczweb/wsPool"
	"github.com/gogo/protobuf/proto"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"
	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

func main() {
	go wsClient("1001_1_2")
	select {

	}
}



func wsClient(id string) {
	flag.Parse()
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	log.Printf("connecting to %s", u.String())
	head := http.Header{}
	log.Printf("connecting info: %s",id)
	head.Add("Sec-Websocket-Protocol", id)
	c, _, err := websocket.DefaultDialer.Dial(u.String(), head)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()
	ping := make(chan int)
	c.SetPingHandler(func(appData string) error {
		ping<-1;
		return nil
	})

	done := make(chan struct{})

	go func() {
		ticker1 := time.NewTicker(10*time.Second)
		defer func() {
			ticker1.Stop()
			close(done)
		}()
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Printf("read:%s", err.Error())
				return
			}
			log.Printf("recv: %s", message)
			select{
			case  <-ticker1.C:
				return
			}
		}
	}()

	ticker := time.NewTicker(time.Second)

	defer ticker.Stop()

	for {
		select {
		case <-done:
			//重新连接
			go wsClient(id)
			return
		case  <-ticker.C:
			msg:=&wsPool.SendMsg{
				ToClientId:"12",
				FromClientId:"13",
				Msg:"test"+time.Now().String(),
				Channel:[]string{"1","2"},
			}
			m,_:=proto.Marshal(msg)
			err := c.WriteMessage(websocket.BinaryMessage, m)
			if err != nil {
				log.Printf("write:%s", err.Error())
				return
			}
		case <-interrupt:
			log.Printf("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Printf("write close:%s", err.Error())
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		case <-ping:
			err := c.WriteMessage(websocket.PongMessage, nil)
			if err != nil {
				log.Printf("write pong:%s", err.Error())
				return
			}
		}
	}
}
