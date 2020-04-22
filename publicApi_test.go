package wsPool

import (
	"log"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	InitWsPool(func(err interface{}) {
		log.Println(err)
	})

	for i:=0;i<1000 ;i++  {
		go func() {
			conf:=&Config{
				Id:string(i),
				Type:"robot",
				Channel:[]string{"1","2","3"},
				Goroutine:10,
			}
			connOjb:=NewClient(conf)
			connOjb.OnOpen(func() {
				log.Println("用户ID",connOjb.Id,"连接成功！")
			})
			//ws连接的事件侦听
			connOjb.OnMessage(func(msg *SendMsg) {
				log.Println(connOjb.Id,"收到消息",msg.GetCmd())
			})

			connOjb.OnPing(func() {
				log.Println("收到连接的Ping",connOjb.Id)
				//cache.PageApiPool.Remove(connOjb.Id)
			})
			connOjb.OnPong(func() {
				log.Println("收到连接的Pong",connOjb.Id)
				//cache.PageApiPool.Remove(connOjb.Id)
			})
			connOjb.OnClose(func() {
				log.Println("连接己经关闭",connOjb.Id)
			})
			connOjb.OnError(func(err error) {
				log.Println("连接",connOjb.Id,"错误信息：",err)
			})

			connOjb.OpenClient(nil,nil,nil)
			time.Sleep(10*time.Second)
			connOjb.Close()
		}()

	}

}