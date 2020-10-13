// Copyright 2015 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package main

import (
	"flag"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"time"
)

var addr = flag.String("addr", "192.168.0.8:8081", "http service address")

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	for i:=1;i<5001 ;i++  {
		time.Sleep(10*time.Millisecond)
		go wsClient(fmt.Sprintf("%d_1_3",i))
	}
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
	defer func() {
		c.Close()
		//重新连接
		time.Sleep(time.Duration(5)*time.Second)
		go wsClient(id)
	}()
	ping := make(chan int)
	c.SetPingHandler(func(appData string) error {
		ping<-1;
		return nil
	})

	done := make(chan struct{})
	//t:=grand.N(30,90)
	go func() {
		//ticker1 := time.NewTicker(time.Duration(t)*time.Second)
		defer func() {
			//ticker1.Stop()
			close(done)
		}()
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Printf("read:%s,%s", id,err.Error())
				return
			}
			log.Printf("recv:id===>%s, %s",id, message)
			/*select{
				case  <-ticker1.C:
					//return
			}*/
		}
	}()

	ticker := time.NewTicker(time.Second)
	i:=1
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case  <-ticker.C:
			i++
			msg:=id+"==="+string(i)
			err := c.WriteMessage(websocket.TextMessage, []byte(msg))
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
			//return
		}
	}
}

