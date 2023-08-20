package connector

import (
	"io"
	"log"
	"net"
)

type tcpType struct {
	listener net.Listener
	Addr     net.Addr
}

type tcpConn struct {
	conn net.Conn
	in   chan message
	out  chan message
}

func NewTcp() tcpType {
	tcp := tcpType{}
	return tcp
}

func (tcp *tcpType) Listen(addr string) (chan message, chan message, error) {
	control := make(chan message)
	notify := make(chan message)

	l, err := net.Listen("tcp", addr)
	if err != nil {
		return control, notify, err
	}
	tcp.listener = l
	tcp.Addr = l.Addr()

	go tcp.doListen(control, notify)

	return control, notify, nil
}

func (tcp *tcpType) doListen(control chan message, notify chan message) {
	defer tcp.listener.Close()

	for {
		conn, err := tcp.listener.Accept()
		if err != nil {
			log.Print(err)
		}

		msg := NewConnectionMessage{}
		msg.In = make(chan message)
		msg.Out = make(chan message)
		notify <- msg

		c := tcpConn{}
		c.conn = conn
		c.in = msg.In
		c.out = msg.Out

		go c.connectionInputHandler()
		go c.connectionOutputHandler()
	}
}

func (c *tcpConn) connectionOutputHandler() {
	defer close(c.out)

	b := make([]byte, 65535) // Largest possible TCP payload
	n, err := io.ReadAtLeast(c.conn, b, 1)
	for err == nil && n > 0 {
		c.out <- TextMessage{Text: string(b[:n])}
		n, err = io.ReadAtLeast(c.conn, b, 1)
	}
	if err != nil {
		if err == io.EOF {
			c.out <- DisconnectMessage{}
		} else {
			c.out <- ErrorMessage{Err: err}
		}
	}
}

func (c *tcpConn) connectionInputHandler() {
	defer c.conn.Close()
	for {
		m, ok := <-c.in
		if !ok {
			return
		}

		switch m.(type) {
		case TextMessage:
			txtMsg := m.(TextMessage)
			b := []byte(txtMsg.Text)
			n, err := c.conn.Write(b)
			if err != nil || n != len(b) {
				log.Print("Could not write full message out of socket")
				return
			}
		case DisconnectMessage:
			return
		default:
			log.Print("Unknown message type: " + m.Type())
		}
	}
}
