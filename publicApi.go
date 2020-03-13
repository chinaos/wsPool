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
		send: make(chan []byte, 256),
		ping: make(chan int, 128),
		IsClose:true,
	}
	client.OnError(nil)
	client.OnOpen(nil)
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
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	c.send<-data
	return nil
}


/*
广播消息每次只能指定一个频道并且不能与ToClientId同时存在。
并且只针对频道内的连接进行处理
*/
func (c *Client) Broadcast(msg *SendMsg) error {
	if  len(msg.Channel)==0 {
		return errors.New("发送消息的消息体中未指定Channel频道！")
	}
	for client,ok:=range c.hub.clients  {
		if !ok {
			continue
		}
		if client.IsClose {
			continue
		}
		for _,ch:=range msg.Channel  {
			if searchStrArray(client.channel,ch){
				msg.ToClientId=client.Id
				err := client.Send(msg)
				if err != nil {
					return errors.New("连接ID："+client.Id+"广播消息出现错误："+ err.Error())
				}
			}
		}

	}

	return nil
}




