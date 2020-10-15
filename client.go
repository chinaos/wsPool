// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wsPool

import (
	"errors"
	"gitee.com/rczweb/wsPool/util/grpool"
	"gitee.com/rczweb/wsPool/util/rwmutex"
	"github.com/gorilla/websocket"
	"net/http"
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
	//ReadBufferSize:  1024 * 1024,
	//WriteBufferSize: 1024 * 1024,
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
	Goroutine int //每个连接开启的go程数里 默认为10
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

type sendMessage struct {
	MessageType int //消息类型1为string 2为[]byte
	Msgbytes []byte //byte消息内容
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
	CloseTime int64 //连接断开的时间
	channel []string //连接注册频道类型方便广播等操作。做为一个数组存储。因为一个连接可以属多个频道
	// Buffered channel of outbound messages.
	grpool *grpool.Pool
	sendCh chan sendMessage
	sendPing chan int
	ping chan int //收到ping的存储管道，方便回复pong处理
	ticker  *time.Ticker //定时发送ping的定时器
	onError func(error)
	onOpen func()  //连接成功的回调
	onPing func()  //收到ping
	onPong func()  //收到pong
	onMessage func([]byte)
	onMessageString func(string)
	onClose func()
	mu    rwmutex.RWMutex //读写锁对象
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
	}()
	for {
		if c.IsClose {
			return
		}
			messageType, message, err := c.conn.ReadMessage()
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
					//c.closeChan<-true;
					return
				}
				c.onError(errors.New("连接ID："+c.Id+"ReadMessage other error:"+err.Error()))
				//c.closeChan<-true;
				return
			}
			c.conn.SetReadDeadline(time.Now().Add(pongWait))

		switch messageType {
			case websocket.TextMessage:
				c.grpool.Add(func() {
					c.onMessageString(string(message))
				})
			case websocket.BinaryMessage:
				c.grpool.Add(func() {
					c.onMessage(message)
				})

		}


	}
}



// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	defer func() {
		c.IsClose=true
		dump()
	}()
	for {
		if c.IsClose {
			return
		}
		select {
		case message, ok := <-c.sendCh:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				//说明管道己经关闭
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				//glog.Error("连接ID："+c.Id,"wsServer发送消息失败,一般是连接channel已经被关闭：(此处服务端会断开连接，使客户端能够感知进行重连)")
				return
			}
			err:=c.conn.WriteMessage(message.MessageType,message.Msgbytes)
			if err != nil {
				c.onError(errors.New("连接ID："+c.Id+"发送消息失败！连接中断"+err.Error()))
				return
			}
			c.lastSendTime = time.Now()
			// Add queued chat messages to the current websocket message.
			n := len(c.sendCh)
			if(n>0){
				for i := 0; i < n; i++ {
					m:= <-c.sendCh
					err:=c.conn.WriteMessage(m.MessageType,m.Msgbytes)
					if err != nil {
						c.onError(errors.New("连接ID："+c.Id+"写上次连接未发送的消息消息进写入IO错误！连接中断"+err.Error()))
						return
					}
				}
			}


		case p, ok :=<-c.sendPing: //定时发送ping
			if !ok {
				continue
			}
			if p==1 {
				c.conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					c.onError(errors.New("连接ID：" + c.Id + "关闭写入IO对象出错，连接中断" + err.Error()))
					return
				}
			}
		case p, ok :=<-c.ping:
			if !ok {
				continue
			}
			if p==1 {
				c.conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := c.conn.WriteMessage(websocket.PongMessage, nil); err != nil {
					c.onError(errors.New("回复客户端PongMessage出现异常:"+err.Error()))
					return
				}
			}
		}
	}
}




func (c *Client) send(msg sendMessage)  {
	if c.IsClose{
		c.onError(errors.New("连接"+c.Id+",连接己在关闭，消息发送失败"))
		return
	}
	timeout := time.NewTimer(time.Millisecond * 800)
	select {
	case c.sendCh<-msg:
		return
	case <-timeout.C:
		c.onError(errors.New("sendCh消息管道blocked,写入消息超时"))
		return
	}
	//c.sendCh<-msg
}



func (c *Client) close() {
	c.IsClose=true
	c.conn.Close()
	//触发连接关闭的事件回调
	c.onClose() //先执行完关闭回调，再请空所有的回调
	c.OnError(nil)
	c.OnOpen(nil)
	c.OnPing(nil)
	c.OnPong(nil)
	c.OnMessage(nil)
	c.OnMessageString(nil)
	c.OnClose(nil)
	c.hub.unregister<-c.Id
}


