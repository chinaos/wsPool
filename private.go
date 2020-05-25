package wsPool

import "github.com/gogo/protobuf/proto"

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
	message.MsgId = msg.MsgId
	message.CmdData = msg.CmdData
	message.Status = msg.Status
	message.Msg = msg.Msg
	message.CallbackMsg = msg.CallbackMsg
	message.Cmdkey = msg.Cmdkey
	message.Priority = msg.Priority
	message.PageKey = msg.PageKey
	message.Channel = msg.Channel
	message.PageId = msg.PageId
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
	message.MsgId = msg.MsgId
	message.CmdData = msg.CmdData
	message.Status = msg.Status
	message.Msg = msg.Msg
	message.Cmdkey = msg.Cmdkey
	message.Priority = msg.Priority
	message.PageKey = msg.PageKey
	message.Channel = msg.Channel
	message.PageId = msg.PageId
	message.Desc = msg.Desc
	message.CallbackMsg = msg.CallbackMsg
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
