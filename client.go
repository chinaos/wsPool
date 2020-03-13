// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wsPool

import (
	"errors"
	"github.com/gorilla/websocket"
	"net/http"
	"sync"
	"time"
)


const (
// Time allowed to read the next pong message from the peer.
 	pongWait = 60 * time.Second

// Send pings to peer with this period. Must be less than pongWait.
	 pingPeriod =(pongWait * 9) / 10 //1 * time.Second//(pongWait * 9) / 10
//var pingPeriod =27* time.Second //1 * time.Second//(pongWait * 9) / 10

	// Time allowed to write a message to the peer.
	writeWait = 30 * time.Second
	// Maximum message size allowed from peer.
	maxMessageSize = 1024 * 1024 * 20
)


var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	// 默认允许WebSocket请求跨域，权限控制可以由业务层自己负责，灵活度更高
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

/*连接参数结构体*/
type Config struct {
	Id              string    //标识连接的名称
	Type            string    //连接类型或path
	Channel []string //连接注册频道类型方便广播等操作。做为一个数组存储。因为一个连接可以属多个频道
}

type RuntimeInfo struct {
	Id              string    //标识连接的名称
	Type            string    //连接类型或path
	Ip string
	Channel []string //连接注册频道类型方便广播等操作。做为一个数组存储。因为一个连接可以属多个频道
	OpenTime time.Time //连接打开时间
	LastReceiveTime time.Time //最后一次接收到数据的时间
	LastSendTime    time.Time //最后一次发送数据的时间
}


// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *hub
	// The websocket connection.
	conn *websocket.Conn
	types            string    //连接类型或path
	openTime time.Time //连接打开时间
	lastReceiveTime time.Time //最后一次接收到数据的时间
	lastSendTime    time.Time //最后一次发送数据的时间
	Id              string    //标识连接的名称
	IsClose bool   //连接的状态。true为关闭
	channel []string //连接注册频道类型方便广播等操作。做为一个数组存储。因为一个连接可以属多个频道
	// Buffered channel of outbound messages.
	*sync.Mutex               // 锁，主要是了为解决并发调用socket连接的问题
	send chan []byte
	ping chan int //收到ping的存储管道，方便回复pong处理
	ticker  *time.Ticker //定时发送ping的定时器
	onError func(error)
	onOpen func()  //连接成功的回调
	onMessage func(*SendMsg)
	onClose func()
}


// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
		dump()
		c.close()
		c.hub.unregister<-c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(str string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait));
		return nil
	})
	c.conn.SetPingHandler(func(str string) error {
		c.ping<-1;
		/*if err := c.conn.WriteMessage(websocket.PongMessage, nil); err != nil {
			c.onError(errors.New("回复客户端PongMessage出现异常:"+err.Error()))
		}*/
		return nil
	})
	/*c.conn.SetCloseHandler(func(code int, str string) error {
		//收到客户端连接关闭时的回调
		glog.Error("连接ID："+c.Id,"SetCloseHandler接收到连接关闭状态:",code,"关闭信息：",str)
		return nil
	})*/

	for {
		if c.IsClose {
			break
		}
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseAbnormalClosure,
				websocket.CloseGoingAway,
				websocket.CloseProtocolError,
				websocket.CloseUnsupportedData,
				websocket.CloseNoStatusReceived,
				websocket.CloseAbnormalClosure,
				websocket.CloseInvalidFramePayloadData,
				websocket.ClosePolicyViolation,
				websocket.CloseMessageTooBig,
				websocket.CloseMandatoryExtension,
				websocket.CloseInternalServerErr,
				websocket.CloseServiceRestart,
				websocket.CloseTryAgainLater,
				websocket.CloseTLSHandshake ) {
				c.onError(errors.New("连接ID："+c.Id+"ReadMessage Is Unexpected Close Error:"+err.Error()))
				break;
			}
			c.onError(errors.New("连接ID："+c.Id+"ReadMessage other error:"+err.Error()))
			break
		}
		c.readMessage(message)
	}
}

// 单个连接接收消息
func (c *Client) readMessage(data []byte) {
	c.lastReceiveTime = time.Now()
	message, err := unMarshal(data)
	if err != nil {
		c.onError(errors.New("接收数据ProtoBuf解析失败！！连接ID："+c.Id+"原因："+err.Error()))
		return
	}
/*	//ToClientId与Channel不能同时存在！！！注意！！！！
	if message.ToClientId!="" {
		Send(message)
	}
	//ToClientId与Channel不能同时存在！！！注意！！！！
	if message.Channel!="" {
		Broadcast(message)
	}*/

	//收到消息触发回调
	if c.onMessage!=nil {
		go c.onMessage(message)
	}

}


// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	c.ticker = time.NewTicker(pingPeriod)
	defer func() {
		dump()
		c.ticker.Stop()
		//c.Close()
		c.conn.Close()
	}()
	for {
		if c.IsClose {
			break
		}
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				//c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				//glog.Error("连接ID："+c.Id,"wsServer发送消息失败,一般是连接channel已经被关闭：(此处服务端会断开连接，使客户端能够感知进行重连)")
				return
			}
			w, err := c.conn.NextWriter(websocket.BinaryMessage)
			if err != nil {
				return
			}
			c.lastSendTime = time.Now()
			_,err=w.Write(message)
			if err != nil {
				c.onError(errors.New("连接ID："+c.Id+"写消息进写入IO错误！连接中断"+err.Error()))
				return
			}
			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				_,err=w.Write(<-c.send)
				if err != nil {
					c.onError(errors.New("连接ID："+c.Id+"写上次连接未发送的消息消息进写入IO错误！连接中断"+err.Error()))
					return
				}
			}
			//关闭写入io对象
			if err := w.Close(); err != nil {
				c.onError(errors.New("连接ID："+c.Id+"关闭写入IO对象出错，连接中断"+err.Error()))
				return
			}

		case <-c.ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))

			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.onError(errors.New("连接ID："+c.Id+"关闭写入IO对象出错，连接中断"+err.Error()))
				return
			}
		case <-c.ping:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PongMessage, nil); err != nil {
				c.onError(errors.New("回复客户端PongMessage出现异常:"+err.Error()))
			}
		}
	}
}

func (c *Client) close() {
	c.IsClose=true
	//触发连接关闭的事件回调
	c.onClose()
}

//服务主动关闭连接
func (c *Client) Close() {
	c.close()
	c.ticker.Stop()
	c.conn.Close()
	c.hub.unregister<-c
}
