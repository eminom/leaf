package network

type Processor interface {
	// must goroutine safe
	DoRoute(msg interface{}, ud interface{}) error
	// must goroutine safe
	Unmarshal(data []byte) (interface{}, error)
	// must goroutine safe
	Marshal(msg interface{}) ([][]byte, error)
}
