# wsPool

#### 介绍
golang websocket 连接池

基于gorilla/websocket和protobuf实现

同时支持各种类型的数据交互


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

### 基层的protobuf格式
除 toClientId,fromClientId,channel外 其它都可以随意定议，增减都可以。

```$xslt

message SendMsg {
  int32 cmd =1;
  int64 timestamp  = 2;
  string fromClientId =3;  //指令消息的来源。发送者的连接ID
  string toClientId = 4;  //指令消息的接收者。发送给对应的客户端连接ID
  bytes cmdData =5;  //对应指令的CmdData1的protobuf的message
  string msgId=6;
  int32 status=7;  //消息发送响应状态
  string callbackMsg=8; //消息发送响应内容
  string cmdkey=9;  //用于区分同一cmd多条指令的key 方便api调用针对同一指令不同回调的处理
  int32 priority=10; //用于处理指令队列的优先级的权重值
  int32 localId = 11; //客户端标识消息的id，主要区分相同cmd的不同消息，方便收到回复分发处理等效果
  string pageKey = 12; //用于页面发送指令和接收的指令相对应 如发送cmd2003和收到的cmd1000对应
  repeated string channel = 13; //指定需要广播的频道，可以指定一个或多个频道
  string pageId = 14;  //用于前端存储处理
  string msg =15; //一般用于json数据传递
  string Desc=16; //消息介绍内容，或其它数据
}

```
作者很懒惰！！

其它看源码和例子，有些注释，很简单 ！