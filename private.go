package wsPool


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
