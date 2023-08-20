package connector

import (
	"io"
	"log"
	"net"
)

type tcpListen struct {
	id       string
	listener net.Listener
	Addr     net.Addr
	control  chan message
	notify   chan message
}

func (listen tcpListen) Id() string            { return listen.id }
func (listen tcpListen) Control() chan message { return listen.control }
func (listen tcpListen) Notify() chan message  { return listen.notify }

type tcpConn struct {
	id       string
	conn     net.Conn
	fromConn chan message
	toConn   chan message
}

func (tcp tcpConn) Id() string             { return tcp.id }
func (tcp tcpConn) FromConn() chan message { return tcp.fromConn }
func (tcp tcpConn) ToConn() chan message   { return tcp.toConn }
func (tcp tcpConn) RemoteAddr() net.Addr   { return tcp.conn.RemoteAddr() }

func NewTcpListen(id string, addr string) (tcpListen, error) {
	listen := tcpListen{}

	l, err := net.Listen("tcp", addr)
	if err != nil {
		return listen, err
	}
	listen.listener = l
	listen.Addr = l.Addr()
	listen.id = id + "-TCP-" + listen.Addr.String()
	listen.control = make(chan message)
	listen.notify = make(chan message)

	go listen.doListen()

	return listen, nil
}

func (listen *tcpListen) doListen() {
	defer listen.listener.Close()

	for {
		conn, err := listen.listener.Accept()
		if err != nil {
			log.Print(err)
		}

		c := tcpConn{}
		c.conn = conn
		c.id = listen.id + "-" + conn.RemoteAddr().String()
		c.fromConn = make(chan message)
		c.toConn = make(chan message)

		msg := NewConnectionMessage{}
		msg.Conn = c
		listen.notify <- msg

		go c.connectionInputHandler()
		go c.connectionOutputHandler()
	}
}

func (c *tcpConn) connectionOutputHandler() {
	defer close(c.fromConn)

	b := make([]byte, 65535) // Largest possible TCP payload
	n, err := io.ReadAtLeast(c.conn, b, 1)
	for err == nil && n > 0 {
		c.fromConn <- TextMessage{Text: string(b[:n])}
		n, err = io.ReadAtLeast(c.conn, b, 1)
	}
	if err != nil {
		if err == io.EOF {
			c.fromConn <- DisconnectMessage{}
		} else {
			c.fromConn <- ErrorMessage{Err: err}
		}
	}
}

func (c *tcpConn) connectionInputHandler() {
	defer c.conn.Close()
	for {
		m, ok := <-c.toConn
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
