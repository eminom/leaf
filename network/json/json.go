package json

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"github.com/name5566/leaf/chanrpc"
	"github.com/name5566/leaf/log"
)

// type MsgHandler func([]interface{})

type MsgInfo struct {
	msgType    reflect.Type        // 完整的类型
	msgRouter  *chanrpc.Server     // 路由？完整声明我还没看到
	//msgHandler MsgHandler          // MsgHandler ??? 哈哈哈 （如下. 通过type定义. 函数指针)
}

type Processor struct {
	msgInfo map[string]*MsgInfo
}

func NewProcessor() *Processor {
	p := new(Processor)
	p.msgInfo = make(map[string]*MsgInfo)
	return p
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) Register(msg interface{}) {
	msgType := reflect.TypeOf(msg)
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		log.Fatal("json message pointer required")
	}
	msgID := msgType.Elem().Name()
	if msgID == "" {
		log.Fatal("unnamed json message")
	}
	if _, ok := p.msgInfo[msgID]; ok {
		log.Fatal("message %v is already registered", msgID)
	}
	i := new(MsgInfo)
	i.msgType = msgType
	p.msgInfo[msgID] = i
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) SetRouter(msg interface{}, msgRouter *chanrpc.Server) {
	msgType := reflect.TypeOf(msg)
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		log.Fatal("json message pointer required")
	}
	msgID := msgType.Elem().Name()
	ent, ok := p.msgInfo[msgID]
	if !ok {
		log.Fatal("message %v not registered", msgID)
	}
	ent.msgRouter = msgRouter
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
//~ It is useless in this way.
/*
func (p *Processor) SetHandler(msg interface{}, msgHandler0 MsgHandler) {
	msgType := reflect.TypeOf(msg)
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		log.Fatal("json message pointer required")
	}
	msgID := msgType.Elem().Name()
	entM, ok := p.msgInfo[msgID]
	if !ok {
		log.Fatal("message %v not registered", msgID)
	}
	entM.msgHandler = msgHandler0
}*/

// goroutine safe
//~ 
func (p *Processor) DoRoute(msg interface{}, ud interface{}) error {
	msgType := reflect.TypeOf(msg)
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		return errors.New("json message pointer required")
	}
	msgID := msgType.Elem().Name()
	i, ok := p.msgInfo[msgID]
	if !ok {
		return fmt.Errorf("message %v not registered", msgID)
	}
	// if i.msgHandler != nil {
	// 	//~ Handler. 不切换线程.
	// 	i.msgHandler([]interface{}{msg, userData})
	// }
	if i.msgRouter != nil {
		//~ 这里的Go...可能就要切换线程了. 
		//~ Switching of goroutine might happen in here.
		i.msgRouter.DoDispatch(msgType, msg, ud) //~ The type(key of the map)
	}
	return nil  //~ Mark of no errors.
}

// goroutine safe
func (p *Processor) Unmarshal(data []byte) (interface{}, error) {
	var m map[string]json.RawMessage
	err := json.Unmarshal(data, &m)
	if err != nil {
		return nil, err
	}
	if len(m) != 1 {
		return nil, errors.New("invalid json data")
	}  //~ And there is only one. 
	for msgID, data := range m {
		i, ok := p.msgInfo[msgID]
		if !ok {
			return nil, fmt.Errorf("message %v not registered", msgID)
		}
		// msg
		msg := reflect.New(i.msgType.Elem()).Interface()
		err := json.Unmarshal(data, msg)
		return msg, err  //分别对应msg本身和error. json.Unmarshal的到的msg的引用从而inplace fix
	}
	panic("bug")  //按照逻辑这是无法达到的部分. But in order to suppress the complains from compiler.
}

// goroutine safe
func (p *Processor) Marshal(msg interface{}) ([][]byte, error) {
	msgType := reflect.TypeOf(msg)
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		return nil, errors.New("json message pointer required")
	}
	msgID := msgType.Elem().Name()
	if _, ok := p.msgInfo[msgID]; !ok {
		return nil, fmt.Errorf("message %v not registered", msgID)
	}
	//  data
	/// 从名字构筑这个JSON.
	//~ 例如: 若我是type Hello struct {
	//~    Name string 
	//  }
	//~  那么我们得到一个{"Hello":{"Name":"nombre"}}
	m := map[string]interface{}{msgID: msg}
	data, err := json.Marshal(m)
	return [][]byte{data}, err
}
