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
	tcp := NewTcp()
	_, notify, err := tcp.Listen("0.0.0.0:0")
	if err != nil {
		t.Errorf("Received an error (%s) from listen", err)
	}
	addr := tcp.Addr.String()
	re, err := regexp.Compile("^(.*):")
	if err != nil {
		t.Errorf("Received a regex compile error (%s)", err)
	}
	portStr := re.ReplaceAllString(addr, "")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Errorf("Received a integer conversion error (%s)", err)
	}

	if port == 0 {
		t.Error("Listening port invalid")
	}

	outbound, err := net.Dial("tcp", addr)
	assert.Equal(t, nil, err, "Error from net.Dial()")

	m, ok := <-notify
	assert.True(t, ok, "Notify channel ok")

	connection := m.(NewConnectionMessage)
	outbound.Write([]byte("Hello"))
	m, ok = <-connection.Out
	assert.True(t, ok, "Channel open")
	assert.Equal(t, "Hello", m.(TextMessage).Text)

	connection.In <- TextMessage{Text: "Foo"}
	b := make([]byte, 65535)
	n, err := outbound.Read(b)
	assert.Equal(t, nil, err, "Outobund read has no error")
	assert.Equal(t, 3, n, "Read right number of bits")
	assert.Equal(t, "Foo", string(b[:n]), "Foo is returned")

	outbound.Close()
	m, ok = <-connection.Out
	assert.True(t, ok, "Channel open")
	assert.IsType(t, DisconnectMessage{}, m, "Disconnect received")

	_, ok = <-connection.Out
	assert.False(t, ok, "Channel closed")
}
