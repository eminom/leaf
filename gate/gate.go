

package gate

import (
	"github.com/name5566/leaf/chanrpc"
	"github.com/name5566/leaf/log"
	"github.com/name5566/leaf/network"
	"reflect"
	"time"
)

//~ Try to fit into the interface of `Module'
//~ And three methods must be implemented.
type Gate struct {
	MaxConnNum      int
	PendingWriteNum int
	MaxMsgLen       uint32
	Processor       network.Processor
	AgentChanRPC    *chanrpc.Server

	// websocket
	WSAddr      string
	HTTPTimeout time.Duration

	// tcp
	TCPAddr      string
	LenMsgLen    int
	LittleEndian bool
}

// func genAgentMaker(gate *Gate) network.Agent{
// 	return func()
// }

func (gate *Gate) _createWSServer() *network.WSServer {
	if gate.WSAddr != "" {
		log.Debug("Creating websocket server for now:%v", gate.WSAddr)
		wsServer := new(network.WSServer)
		wsServer.Addr = gate.WSAddr
		wsServer.MaxConnNum = gate.MaxConnNum
		wsServer.PendingWriteNum = gate.PendingWriteNum
		wsServer.MaxMsgLen = gate.MaxMsgLen
		wsServer.HTTPTimeout = gate.HTTPTimeout
		wsServer.NewAgent = func(conn *network.WSConn) network.Agent {
			a := &xagent{conn: conn, gate: gate}
			if gate.AgentChanRPC != nil {
				gate.AgentChanRPC.DoDispatch("NewAgent", a)
			}
			return a
		}
		return wsServer
	}
	return nil
}

func (gate *Gate) _createTCPServer() *network.TCPServer {
	if gate.TCPAddr != "" {
		log.Debug("Creating TCP server for now:%v", gate.TCPAddr)
		tcpServer := new(network.TCPServer)
		tcpServer.Addr = gate.TCPAddr
		tcpServer.MaxConnNum = gate.MaxConnNum
		tcpServer.PendingWriteNum = gate.PendingWriteNum
		tcpServer.LenMsgLen = gate.LenMsgLen
		tcpServer.MaxMsgLen = gate.MaxMsgLen
		tcpServer.LittleEndian = gate.LittleEndian
		tcpServer.NewAgent = func(conn *network.TCPConn) network.Agent {
			a := &xagent{conn: conn, gate: gate}
			if gate.AgentChanRPC != nil {
				gate.AgentChanRPC.DoDispatch("NewAgent", a)
			}
			return a
		}
		return tcpServer
	}	
	return nil
}


//~ Aim to implement Module's interface of `Run'
func (gate *Gate) Run(clozeSig chan bool) {
	if wsServer := gate._createWSServer(); wsServer!= nil {
		wsServer.Start()
		defer func(){
			log.Debug("Closing of ws-server...")
			wsServer.Close()	//~ Watch 
		}()
	}
	if tcpServer := gate._createTCPServer(); tcpServer != nil {
		tcpServer.Start()
		defer func(){
			log.Debug("Closing of tcp-server")
			tcpServer.Close() //~ Is is abusive ?? No.
		}()
	}
	<- clozeSig
}

//~ 此处的Gate未实现Module的OnInit接口。 要派生类中实现.
func (gate *Gate) OnInit() {
 	log.Debug("Gate's OnInit ??")
 	panic("You shall override Gate's OnInit")
}
func (gate *Gate) OnDestroy() {
	log.Debug("Gate's OnDestroy ?? The default implementation from Gate")
}
//~ 有且仅有Run实现了. 

////////////////
//~ Not the one exported(Distinguished from A-gent)
//~     network's Agent is implemented( Run/OnClose)
//~ And gate's Agent is also implemented. (WriteMsg and so on)
type xagent struct {
	conn     network.Conn  //~ This si combination. 
	gate     *Gate         //~ 
	//userData interface{}   // .Kind() == reflect.Ptr. For now this is useless.
}

//~ Depends on Message Processor. 
//~ interface: network.Run
func (a *xagent) Run() {
	for {
		data, err := a.conn.ReadMsg()
		if err != nil {
			log.Debug("read message: %v", err)
			break
		}
		if a.gate.Processor != nil {
			msg, err := a.gate.Processor.Unmarshal(data)
			if err != nil {
				log.Debug("unmarshal message error: %v", err)
				break
			}
			//~ And the message will be dispatched through `Do-Dispatch'
			//~ The message is mapped to chanrpc.Server (to which `Do-Dispatch' belongs)
			//~ And the across goroutine calls are made in here.
			//~ Shall we combine the two procedures(Unmarshal and DoRoute(redistribute))
			err = a.gate.Processor.DoRoute(msg, a)
			if err != nil {
				log.Debug("route message error: %v", err)
				break
			}
		}
	}
}

//~ interface : network.Agent
func (a *xagent) OnClose() {
	if a.gate.AgentChanRPC != nil {
		err := a.gate.AgentChanRPC.Call0("CloseAgent", a)
		if err != nil {
			log.Error("chanrpc error: %v", err)
		}
	}
}

func (a *xagent) WriteMsg(msg interface{}) {
	if a.gate.Processor != nil {
		data, err := a.gate.Processor.Marshal(msg)
		if err != nil {
			log.Error("marshal message %v error: %v", reflect.TypeOf(msg), err)
			return
		}
		err = a.conn.WriteMsg(data...)
		if err != nil {
			log.Error("write message %v error: %v", reflect.TypeOf(msg), err)
		}
	}
}

func (a *xagent) Close() {
	a.conn.Close()
}

func (a *xagent) Destroy() {
	a.conn.Destroy()   //~ What is the difference from C-lose() ??
}

// func (a *xagent) UserData() interface{} {
// 	return a.userData
// }

// func (a *xagent) SetUserData(data interface{}) {
// 	a.userData = data
// }
