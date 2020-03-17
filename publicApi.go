package wsPool

import (
	"errors"
	"net/http"
	"sync"
	"time"
)


/*
第一步，实例化连接对像
*/
func NewClient(conf *Config) *Client{
	client := &Client{
		Id:conf.Id,
		types:conf.Type,
		channel:conf.Channel,
		hub: wsSever.hub,
		Mutex: &sync.Mutex{},
		sendCh: make(chan []byte, 2048),
		ping: make(chan int, 2048),
		IsClose:true,
	}
	client.OnError(nil)
	client.OnOpen(nil)
	client.OnPing(nil)
	client.OnPong(nil)
	client.OnMessage(nil)
	client.OnClose(nil)
	client.hub.register <- client
	return client
}



//开启连接
// serveWs handles websocket requests from the peer.
func (c *Client)OpenClient(w http.ResponseWriter, r *http.Request, head http.Header) {
	defer dump();


	conn, err := upgrader.Upgrade(w, r, head)
	if err != nil {
		if c.onError!=nil{
			c.onError(err)
		}
		return
	}
	c.conn=conn
	c.IsClose=false
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(str string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait));
		go c.onPong()
		return nil
	})
	c.conn.SetPingHandler(func(str string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		c.ping<-1;
		/*if err := c.conn.WriteMessage(websocket.PongMessage, nil); err != nil {
			c.onError(errors.New("回复客户端PongMessage出现异常:"+err.Error()))
		}*/
		go c.onPing()
		return nil
	})
	/*c.conn.SetCloseHandler(func(code int, str string) error {
		//收到客户端连接关闭时的回调
		glog.Error("连接ID："+c.Id,"SetCloseHandler接收到连接关闭状态:",code,"关闭信息：",str)
		return nil
	})*/

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go c.writePump()
	go c.readPump()
	c.openTime=time.Now()
	c.onOpen()
}

/*
获取连接对像运行过程中的信息
*/
func (c *Client) GetRuntimeInfo() *RuntimeInfo{
	return &RuntimeInfo{
		Id:c.Id,
		Type:c.types,
		Channel:c.channel,
		OpenTime:c.openTime,
		LastReceiveTime:c.lastSendTime,
		LastSendTime:c.lastSendTime,
		Ip:c.conn.RemoteAddr().String(),
	}
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

/*监听连接对象的连接open成功的事件*/
func (c *Client) OnMessage( h func(msg *SendMsg)){
	if h==nil{
		c.onMessage= func(msg *SendMsg) {

		}
		return
	}
	c.onMessage=h
}

/*监听连接对象的连接open成功的事件*/
func (c *Client) OnClose( h func()){
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
func (c *Client) Send(msg *SendMsg) error  {
	if c.IsClose {
		return errors.New("连接ID："+c.Id+"ws连接己经断开，无法发送消息")
	}
	msg.ToClientId=c.Id
	data, err := marshal(msg)
	if err != nil {
		return errors.New("连接ID："+c.Id+"生成protubuf数据失败！原因：:"+err.Error())
	}
	//log.Debug("连接：" + msg.ToClientId + "开始发送消息！");
	c.send(data)
	return nil
}

//服务主动关闭连接
func (c *Client) Close() {
	c.close()
	c.ticker.Stop()
	c.conn.Close()
	c.hub.unregister<-c
}




/*包级的公开方法*/
/*
// 发送消息 只从连接池中按指定的toClientId的连接对象发送出消息
在此方法中sendMsg.Channel指定的值不会处理
*/
func Send(msg *SendMsg) error {
	//log.Info("发送指令：",msg.Cmd,msg.ToClientId)
	if msg.ToClientId=="" {
		return errors.New("发送消息的消息体中未指定ToClient目标！")
	}
	wsSever.hub.sendByToClientId<-msg
	return nil
}

//通过连接池广播消息，每次广播只能指定一个类型下的一个频道
/*
广播消息每次只能指定一个类型和一个频道
广播消息分两种情况
并且只针对频道内的连接进行处理
*/

func Broadcast(msg *SendMsg) error {
	if  len(msg.Channel)==0 {
		return errors.New("广播消息的消息体中未指定Channel频道！")
	}

/*	m:=&sync.Mutex{}
	m.Lock()
	defer m.Unlock()*/
	wsSever.hub.chanBroadcast<-msg
	return nil
}


/*
全局广播
通过此方法进行广播的消息体，会对所有的类型和频道都进行广播
*/
func BroadcastAll(msg *SendMsg) error {
	data, err := marshal(msg)
	if err != nil {
		return errors.New("全局广播生成protubuf数据失败！原因：:"+err.Error())
	}
	wsSever.hub.broadcast<-data
	return nil
}




