package wsPool

import (
	"errors"
	"gitee.com/rczweb/wsPool/util/grpool"
	"github.com/gorilla/websocket"
	"net/http"
	"time"
)


/*
第一步，实例化连接对像
*/
func NewClient(conf *Config) *Client{
	if conf.Goroutine==0{
		conf.Goroutine=1024
	}
	var client *Client
	oldclient:=wsSever.hub.clients.Get(conf.Id)
	if oldclient!=nil {
		c:=oldclient.(*Client)
		if !c.IsClose {
			c.close()
		}
	}
	client = &Client{
		Id:conf.Id,
		types:conf.Type,
		channel:conf.Channel,
		hub: wsSever.hub,
		sendCh: make(chan sendMessage,4096),

		ping: make(chan int),
		sendPing:make(chan int),
		IsClose:true,
		grpool:grpool.NewPool(conf.Goroutine),
	}
	client.OnError(nil)
	client.OnOpen(nil)
	client.OnPing(nil)
	client.OnPong(nil)
	client.OnMessage(nil)
	client.OnMessageString(nil)
	client.OnClose(nil)
	return client
}



//开启连接
// serveWs handles websocket requests from the peer.
func (c *Client)OpenClient(w http.ResponseWriter, r *http.Request, head http.Header)  {
	defer dump();
	conn, err := upgrader.Upgrade(w, r, head)
	if err != nil {
		if c.onError!=nil{
			c.onError(err)
		}
		return
	}
	r.Close=true
	c.conn=conn
	c.IsClose=false
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(str string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait));
		c.sendPing<-1
		c.grpool.Add(c.onPong)
		return nil
	})
	c.conn.SetPingHandler(func(str string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		c.ping<-1;
		/*if err := c.conn.WriteMessage(websocket.PongMessage, nil); err != nil {
			c.onError(errors.New("回复客户端PongMessage出现异常:"+err.Error()))
		}*/
		c.grpool.Add(c.onPing)
		return nil
	})
	/*c.conn.SetCloseHandler(func(code int, str string) error {
		//收到客户端连接关闭时的回调
		glog.Error("连接ID："+c.Id,"SetCloseHandler接收到连接关闭状态:",code,"关闭信息：",str)
		return nil
	})*/

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	//连接开启后瑞添加连接池中
	c.openTime=time.Now()
	c.hub.register <- c
	go c.writePump()
	go c.readPump()
	//c.sendPing<-1
	//go c.tickers()
	c.onOpen()
}


func (c *Client) GetConnectType() string{ //获取连接类型
	return c.types
}
func (c *Client) GetChannel() []string{ //获取连接对象注册的频道
	return c.channel
}
func (c *Client) GetOpenTime() time.Time{ //获取连接打开的时间
	return c.openTime
}
func (c *Client) GetLastReceiveTime() time.Time{ //获取连接最后接收消息的时间
	return c.openTime
}
func (c *Client) GetLastSendTime() time.Time{ //获取连接最后发送消息的时间
	return c.lastSendTime
}
func (c *Client) GetConnectIp() string{ //获取连接客户端ip
	return c.conn.RemoteAddr().String()
}


/*回调添加方法*/

/*监听连接对象的连接open成功的事件*/
func (c *Client) OnOpen(h func()){
	if h==nil{
		c.onOpen= func() {

		}
		return
	}
	c.onOpen=h
}

/*监听连接对象的连接open成功的事件*/
func (c *Client) OnPing(h func()){
	if h==nil{
		c.onPing= func() {

		}
		return
	}
	c.onPing=h
}

/*监听连接对象的连接open成功的事件*/
func (c *Client) OnPong(h func()){
	if h==nil{
		c.onPong= func() {

		}
		return
	}
	c.onPong=h
}

/*
监听连接对象的连接open成功的事件
接收byte类型消息
*/
func (c *Client) OnMessage(h func(msg []byte)){
	if h==nil{
		c.onMessage= func(msg []byte) {

		}
		return
	}
	c.onMessage=h
}

/*
监听连接对象的连接open成功的事件
接收string类型消息
*/
func (c *Client) OnMessageString(h func(msg string)){
	if h==nil{
		c.onMessageString= func(msg string) {

		}
		return
	}
	c.onMessageString=h
}

/*监听连接对象的连接open成功的事件*/
func (c *Client) OnClose(h func()){
	if h==nil{
		c.onClose= func() {

		}
		return
	}
	c.onClose=h
}

/*监听连接对象的错误信息*/
func (c *Client) OnError(h func(err error)){
	if h==nil{
		c.onError= func(err error) {
		}
		return
	}
	c.onError=h
}


// 单个连接发送消息
/*
messageType=1 为string
messageType=2 为[]byte
msg 为消息内容按指定的类型输入
*/
func (c *Client) Send(messageType int,msg interface{}) error  {
	if c.IsClose {
		return errors.New("连接ID："+c.Id+"ws连接己经断开，无法发送消息")
	}
	if messageType>2 {
		return errors.New("连接ID："+c.Id+"消息类型错误！")
	}
	if msg==nil {
		return errors.New("连接ID："+c.Id+"发送的消息内容不能为空！")
	}
	switch messageType {
	case websocket.TextMessage:
		c.send(sendMessage{
			MessageType: messageType,
			Msgbytes: []byte(msg.(string)),
		})
	case websocket.BinaryMessage:
		c.send(sendMessage{
			MessageType: messageType,
			Msgbytes: msg.([]byte),
		})
	}
	return nil
}



//服务主动关闭连接
func (c *Client) Close() {
	c.close()
}

/*
包级的公开方法
所有包级的发送如果连接断开，消息会丢失
*/

//通过连接池广播消息，每次广播只能指定一个类型下的一个频道

/*
广播消息方法
messageType=1 为string
messageType=2 为[]byte
msg 为消息内容按指定的类型输入
channel 为指定频道进行广播可为空，也多个
	如果没有指定channel 是广播所有连接
	注意，channel不能为0
*/


func Broadcast(messageType int,msg interface{},channel ...string) error {

	if messageType>2 {
		return errors.New("广播的消息类型错误！")
	}
	if msg==nil {
		return errors.New("广播发送的消息内容不能为空！")
	}
	var smsg sendMessage
	switch messageType {
	case websocket.TextMessage:
		smsg=sendMessage{
			MessageType: messageType,
			Msgbytes: []byte(msg.(string)),
		}
	case websocket.BinaryMessage:
		smsg=sendMessage{
			MessageType: messageType,
			Msgbytes: msg.([]byte),
		}
	}

	if len(channel)==0 {
		return wsSever.hub.brostcastMsg(broadcastMessage{
			Channel:"0",
			Msg: smsg,
		})
	}else{
		for _, o := range channel {
			return wsSever.hub.brostcastMsg(broadcastMessage{
				Channel:o,
				Msg: smsg,
			})
		}
	}
	return nil
}


/*
通过id获取相应的连接对象
*/
func GetClientById(id string) *Client  {
	c:= wsSever.hub.clients.Get(id)
	if c!=nil {
		return c.(*Client)
	}
	return nil
}



