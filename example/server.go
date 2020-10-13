package main

import (
	"flag"
	"gitee.com/rczweb/wsPool"
	"log"
	"net/http"
	"runtime"
	"strings"
)


var addr = flag.String("addr", "192.168.0.8:8081", "http service address")

func serveHome(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "home.html")
}

var ch =make(chan int,10)
func chfun(i int){
	log.Println("写入管道i的值%d",i)
	//ch<-i
	select {
	case ch<-i:
		return
	default:
		log.Println("管道己经锁定；i的值"+string(i))
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	/*	for i:=0;i<10000 ;i++  {
			go chfun(i)
		}
		close(ch)
*/

	/*for {
		select {
		case i,ok:=<-ch:
			if !ok{
				log.Println("管道己经关闭%d",i)
			}
			log.Println("读取i的值%d",i)

		}
	}
*/





	flag.Parse()
	//初骀化连接池
	wsPool.InitWsPool(func(err interface{}) {
		//接收连接池中的运行时错误信息
		log.Println(err)
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
			Goroutine:32,
		})
		log.Println(client.Id,"实例化连接对象完成")

		//开启连接
		client.OpenClient(w,r,head)
		log.Println(client.Id,"开启连接")

		//连接成功回调
		client.OnOpen(func() {
			log.Printf("连接己开启%s",client.Id)
		})
		i:=1
		//接收消息
		client.OnMessageString(func(msg string) {

				log.Println("OnMessage:收到消息=》",client.Id,msg)
				i++

				/*err:=client.Send(1,"回复客户端消息"+client.Id+string(i))
				if err!=nil {
					log.Println("发送消息出错", err.Error())
				}*/

			err:=wsPool.Broadcast(1,"wsPool.Broadcast"+client.Id+string(i),client.GetChannel()[0])
			if err!=nil {
				log.Println("wsPool.BroadcastAll(msg)", err.Error())
			}

		})
		//连接断开回调
		client.OnClose(func() {
			log.Printf("连接己经关闭%s",client.Id)
		})
		client.OnError(func(err error) {
			log.Printf("连接%s错误信息：%s",client.Id,err.Error())
		})

		client.OnPong(func() {
			//log.Printf("收到连接的Pong:%s",client.Id)
			//cache.PageApiPool.Remove(connOjb.Id)
		})
		client.OnPing(func() {
			//log.Printf("收到连接的Ping:%s",client.Id)
			//cache.PageApiPool.Remove(connOjb.Id)
		})
		r.Close=true
	})
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Printf("ListenAndServe: %s", err.Error())
	}
}




