package wsPool

import (
	"errors"
	"sync"
)
var wsSever *Server

/*连接池的结构体*/
type Server struct {
	hub *hub
	ErrFun func(err interface{}) //用于接收ws连接池内代码运行时错误信息
}


//初始化执行连接池对象
//参数为接收连接池中运行时的一些错误信息的回调方法
func InitWsPool(errfun func(err interface{}) ){
	wsSever = new(Server)
	wsSever.hub = newHub()
	wsSever.ErrFun = errfun
	go wsSever.hub.run()
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

	for client,ok:=range wsSever.hub.clients  {
		if !ok {
			continue
		}
		//找到ToClientId对应的连接对象
		if client.Id==msg.ToClientId{
			if client.IsClose {
				return errors.New("发送消息错误：连接对像："+client.Id+"连接状态异常，连接己经关闭！")
			}
			msg.ToClientId=client.Id
			err := client.Send(msg)
			if err != nil {
				return errors.New("发送消息出错："+ err.Error()+",连接对象id="+client.Id+"。")
			}
			break;
		}
	}

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
		return errors.New("发送消息的消息体中未指定Channel频道！")
	}

	for client, ok := range wsSever.hub.clients {
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

/*
全局广播
通过此方法进行广播的消息体，会对所有的类型和频道都进行广播
*/
func BroadcastAll(msg *SendMsg) error {
	m:=&sync.Mutex{}
	data, err := marshal(msg)
	if err != nil {
		return errors.New("全局广播生成protubuf数据失败！原因：:"+err.Error())
	}
	m.Lock()
	defer m.Unlock()
	wsSever.hub.broadcast<-data
	return nil
}