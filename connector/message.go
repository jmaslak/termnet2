package connector

type message interface {
	Type() MessageType
	TypeString() string
}

type NewConnectionMessage struct {
	Conn Connection
}

type DisconnectMessage struct{}

type DataMessage struct {
	Data []byte
}

type ErrorMessage struct {
	Err error
}

type MessageType int64

const (
	MTDisconnectMessage MessageType = iota
	MTNewConnectionMessage
	MTDataMessage
	MTErrorMessage
)

func (msg DisconnectMessage) Type() MessageType    { return MTDisconnectMessage }
func (msg NewConnectionMessage) Type() MessageType { return MTNewConnectionMessage }
func (msg DataMessage) Type() MessageType          { return MTDataMessage }
func (msg ErrorMessage) Type() MessageType         { return MTErrorMessage }

func (msg DisconnectMessage) TypeString() string    { return "DisconnectMessage" }
func (msg NewConnectionMessage) TypeString() string { return "NewConnectionMessage" }
func (msg DataMessage) TypeString() string          { return "DataMessage" }
func (msg ErrorMessage) TypeString() string         { return "ErrorMessage" }

func NewDataMessage(b []byte) DataMessage {
	return DataMessage{Data: b}
}

func NewDataMessageFromString(s string) DataMessage {
	return NewDataMessage([]byte(s))
}

func (dataMessage DataMessage) String() string {
	return string(dataMessage.Data)
}
