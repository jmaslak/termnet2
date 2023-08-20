package connector

type message interface {
	Type() string
}

type NewConnectionMessage struct {
	Conn Connection
}

type DisconnectMessage struct{}

type TextMessage struct {
	Text string
}

type ErrorMessage struct {
	Err error
}

func (msg DisconnectMessage) Type() string    { return "DisconnectMessage" }
func (msg NewConnectionMessage) Type() string { return "NewConnectionMessage" }
func (msg TextMessage) Type() string          { return "TextMessage" }
func (msg ErrorMessage) Type() string         { return "ErrorMessage" }

func NewTextMessage(s string) TextMessage {
	return TextMessage{Text: s}
}
