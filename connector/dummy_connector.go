package connector

import (
	"strconv"
	"sync"
)

type DummyConnection struct {
	id       string
	fromConn chan message
	toConn   chan message
}

func (dummy DummyConnection) Id() string             { return dummy.id }
func (dummy DummyConnection) FromConn() chan message { return dummy.fromConn }
func (dummy DummyConnection) ToConn() chan message   { return dummy.toConn }

var dummyInstance = 0
var dummyMutex sync.Mutex

func NewDummyConnection(id string) (DummyConnection, error) {
	dummy := DummyConnection{}

	dummyMutex.Lock()
	dummy.id = id + "-Dummy-" + strconv.Itoa(dummyInstance)
	dummyMutex.Unlock()

	dummy.fromConn = make(chan message)
	dummy.toConn = make(chan message)

	return dummy, nil
}

func (dummy DummyConnection) Send(m message) { dummy.fromConn <- m }
func (dummy DummyConnection) Recv() (message, bool) {
	o, err := <-dummy.toConn
	return o, err
}
