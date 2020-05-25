package wsPool

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
	go wsSever.hub.run() //开启服务
	//go wsSever.hub.runbroadcast() //开启广播服务
	go wsSever.hub.ticker() //开启定时服务
}




