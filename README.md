# wsPool

#### 介绍
golang websocket 连接池

支持各种类型的数据交互


#### examples

```$xslt
func main() {
	flag.Parse()
	//初骀化连接池
	wsPool.InitWsPool(func(err interface{}) {
		//接收连接池中的运行时错误信息
		log.Panicln(err)
	})
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		pcol := r.Header.Get("Sec-Websocket-Protocol")
		list:=strings.Split(pcol, "_")
		head := http.Header{}
		head.Add("Sec-Websocket-Protocol", pcol)

		//实例化连接对象
		client:=wsPool.NewClient(&wsPool.Config{
			Id:list[0], //连接标识
			Type:"ws", //连接类型
			Channel:list[1:], //指定频道
		})

		//开启连接
		client.OpenClient(w,r,head)

		//连接成功回调
		client.OnOpen(func() {
			log.Panicln("连接己开启"+client.Id)
		})

		//接收消息
		client.OnMessage(func(msg *wsPool.SendMsg) {
			log.Panicln(msg)
			if msg.ToClientId!="" {
				//发送消息给指定的ToClientID连接
				wsPool.Send(msg)
				//发送消息给当前连接对象
				client.Send(msg)
			}
			if len(msg.Channel)>0{
				//按频道广播，可指定多个频道[]string
				client.Broadcast(msg) //或者 wsPool.Broadcast(msg)
			}
			//或都全局广播，所有连接都进行发送
			wsPool.BroadcastAll(msg)

		})
		//连接断开回调
		client.OnClose(func() {
			log.Panicln("连接己经关闭"+client.Id)
		})
		client.OnError(func(err error) {
			log.Panicln("连接",client.Id,"错误信息：",err)
		})

	})
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}


```
作者很懒惰！！

其它看源码和例子，有些注释，很简单 ！