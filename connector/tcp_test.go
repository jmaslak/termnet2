package connector

import (
	"net"
	"regexp"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTcpListen(t *testing.T) {
	t.Parallel()

	tcp, err := NewTcpListen("0", "")
	assert.Equal(t, nil, err, "Listener does not return error")

	addr := tcp.Addr.String()
	re, err := regexp.Compile("^(.*):")
	assert.Equal(t, nil, err, "Regexp does not return error")
	portStr := re.ReplaceAllString(addr, "")
	port, err := strconv.Atoi(portStr)
	assert.Equal(t, nil, err, "Atoi does not return error")
	assert.Greater(t, port, 0, "Port > 0")

	id := "0-TCP-" + addr
	assert.Equal(t, id, tcp.Id(), "TCP listener ID correct")

	outbound, err := net.Dial("tcp", addr)
	assert.Equal(t, nil, err, "Error from net.Dial()")

	m, ok := <-(tcp.Notify())
	assert.True(t, ok, "Notify channel ok")
	assert.IsType(t, NewConnectionMessage{}, m, "New connection message")

	connectionMsg := m.(NewConnectionMessage)
	assert.IsType(t, NewConnectionMessage{}, connectionMsg, "New connection message")
	assert.IsType(t, tcpConn{}, connectionMsg.Conn, "New connection message")

	id = "0-TCP-" + addr + "-" + connectionMsg.Conn.(tcpConn).RemoteAddr().String()
	assert.Equal(t, id, connectionMsg.Conn.Id(), "TCP listener ID correct")

	outbound.Write([]byte("Hello"))
	m, ok = <-connectionMsg.Conn.FromConn()
	assert.True(t, ok, "Channel open")
	assert.Equal(t, "Hello", m.(TextMessage).Text)

	connectionMsg.Conn.ToConn() <- TextMessage{Text: "Foo"}
	b := make([]byte, 65535)
	n, err := outbound.Read(b)
	assert.Equal(t, nil, err, "Outobund read has no error")
	assert.Equal(t, 3, n, "Read right number of bits")
	assert.Equal(t, "Foo", string(b[:n]), "Foo is returned")

	outbound.Close()
	m, ok = <-connectionMsg.Conn.FromConn()
	assert.True(t, ok, "Channel open")
	assert.IsType(t, DisconnectMessage{}, m, "Disconnect received")

	_, ok = <-connectionMsg.Conn.FromConn()
	assert.False(t, ok, "Channel closed")
}
