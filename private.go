package wsPool

import (
	"errors"
	"github.com/gogo/protobuf/proto"
	"time"
)

/*
client连接对象的私有方法
*/

func dump() {
	rcv_err := recover()
	if rcv_err == nil {
		return
	}
	wsSever.ErrFun(rcv_err)
	//log.Error("运行过程中出现异常，错误信息如下：",rcv_err)
}


/*
如果消息指定发送的目标连接ToClientId的对象与指定广播的频道Channel在同一频道时，只进行广播就行了
消息体格式使用protobuf进行交互
*/
/*type SendMsg struct {
	Cmd          int32
	Timestamp    int64
	FromClientId string  //消息的来源id
	ToClientId   string  //消息指定发送的目标连接id
	CmdData      []byte
	MsgId        string
	Status       int32
	CallbackMsg  string
	Cmdkey       string
	Priority     int32
	PageKey      string
	Channel      []int32  //指定广播的频道
	PageId 		 string
}
*/
// 将发送的对象转为protobuf结果
func marshal(msg *SendMsg) ([]byte, error) {
	message := &SendMsg{}
	message.Cmd = msg.Cmd
	message.Timestamp = msg.Timestamp
	message.FromClientId = msg.FromClientId
	message.ToClientId = msg.ToClientId
	message.CmdData = msg.CmdData
	message.LocalId=msg.LocalId
	message.ServerId=msg.ServerId
	message.Status = msg.Status
	message.Msg = msg.Msg
	message.Priority = msg.Priority
	message.Channel = msg.Channel
	message.Desc = msg.Desc
	data, err := proto.Marshal(message)
	return data, err
}

// 将取到的protobuf的数据转换成发送对象
func unMarshal(data []byte) (*SendMsg, error) {
	msg := &SendMsg{}
	err := proto.Unmarshal(data, msg)
	if err != nil {
		return nil, err
	}
	message := new(SendMsg)
	message.Cmd = msg.Cmd
	message.Timestamp = msg.Timestamp
	message.FromClientId = msg.FromClientId
	message.ToClientId = msg.ToClientId
	message.CmdData = msg.CmdData
	message.Status = msg.Status
	message.LocalId=msg.LocalId
	message.ServerId=msg.ServerId
	message.Msg = msg.Msg
	message.Priority = msg.Priority
	message.Channel = msg.Channel
	message.Desc = msg.Desc
	return message, err
}


 func searchStrArray(arr []string,ch string) bool{
	 result := -1
	 for index, v := range arr{
		 if v == ch {
			 result = index
			 break
		 }
	 }
 	return result!=-1
 }


/*包级的私有方法*/
/*
// 发送消息 只从连接池中按指定的toClientId的连接对象发送出消息
在此方法中sendMsg.Channel指定的值不会处理
*/
func send(msg *SendMsg) error {
	//log.Info("发送指令：",msg.Cmd,msg.ToClientId)
	if msg.ToClientId=="" {
		return errors.New("发送消息的消息体中未指定ToClient目标！")
	}
	if len(wsSever.hub.sendByToClientId)>255{
		return errors.New("发送消息的管道己经写满，请稍后再试！")
	}
	timeout := time.NewTimer(time.Microsecond * 800)

	select {
	case wsSever.hub.sendByToClientId<-msg:
		return nil
	case <-timeout.C:
		return errors.New("sendByToClientId消息管道blocked,写入消息超时")
		/*default:
			c.onError(errors.New("sendCh消息管道blocked,无法写入"))*/
	}
}

//通过连接池广播消息，每次广播只能指定一个类型下的一个频道
/*
广播消息每次只能指定一个类型和一个频道
广播消息分两种情况
并且只针对频道内的连接进行处理
*/

func broadcast(msg *SendMsg) error {
	if  len(msg.Channel)==0 {
		return errors.New("广播消息的消息体中未指定Channel频道！")
	}
	if len(wsSever.hub.chanBroadcast)>255{
		return errors.New("Channel广播消息的管道己经写满，请稍后再试！")
	}
	timeout := time.NewTimer(time.Microsecond * 800)
	select {
	case wsSever.hub.chanBroadcast<-msg:
		return nil
	case <-timeout.C:
		return errors.New("chanBroadcast消息管道blocked,写入消息超时")
		/*default:
			c.onError(errors.New("sendCh消息管道blocked,无法写入"))*/
	}
}


/*
全局广播
通过此方法进行广播的消息体，会对所有的类型和频道都进行广播
*/
func broadcastAll(msg *SendMsg) error {
	if len(wsSever.hub.broadcast)>255{
		return errors.New("广播消息的管道己经写满，请稍后再试！")
	}
	timeout := time.NewTimer(time.Microsecond * 800)
	select {
	case wsSever.hub.broadcast<-msg:
		return nil
	case <-timeout.C:
		return errors.New("broadcast消息管道blocked,写入消息超时")
		/*default:
			c.onError(errors.New("sendCh消息管道blocked,无法写入"))*/
	}
}