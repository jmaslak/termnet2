package connector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTelnet(t *testing.T) {
	t.Parallel()

	dummy, err := NewDummyConnection("0")
	assert.Equal(t, nil, err, "No dummy connection error")

	telnet, err := NewTelnetFilter(dummy)
	assert.Equal(t, nil, err, "No telnet filter error")
	assert.Equal(t, dummy.Id()+"-(telnet)", telnet.Id(), "Telnet ID is proper")

	// Initial sent WILLs
	o, ok := dummy.Recv()
	assert.Equal(t, true, ok, "No dummy receive error")
	assert.IsType(t, DataMessage{}, o, "Data type is proper")
	assert.Equal(t, []byte{255, 251, 0}, o.(DataMessage).Data, "Sent WILL OPT Binary")

	o, ok = dummy.Recv()
	assert.Equal(t, true, ok, "No dummy receive error")
	assert.IsType(t, DataMessage{}, o, "Data type is proper")
	assert.Equal(t, []byte{255, 251, 1}, o.(DataMessage).Data, "Sent WILL OPT Echo")

	o, ok = dummy.Recv()
	assert.Equal(t, true, ok, "No dummy receive error")
	assert.IsType(t, DataMessage{}, o, "Data type is proper")
	assert.Equal(t, []byte{255, 251, 3}, o.(DataMessage).Data, "Sent WILL OPT Suppress Go Ahead")

	// Initial sent Do / Donts
	o, ok = dummy.Recv()
	assert.Equal(t, true, ok, "No dummy receive error")
	assert.IsType(t, DataMessage{}, o, "Data type is proper")
	assert.Equal(t, []byte{255, 253, 0}, o.(DataMessage).Data, "Sent DO OPT Suppress Go Ahead and DONT Echo")

	o, ok = dummy.Recv()
	assert.Equal(t, true, ok, "No dummy receive error")
	assert.IsType(t, DataMessage{}, o, "Data type is proper")
	assert.Equal(t, []byte{255, 254, 1}, o.(DataMessage).Data, "Sent DONT OPT Echo")

	o, ok = dummy.Recv()
	assert.Equal(t, true, ok, "No dummy receive error")
	assert.IsType(t, DataMessage{}, o, "Data type is proper")
	assert.Equal(t, []byte{255, 253, 3}, o.(DataMessage).Data, "Sent DO OPT Suppress Go Ahead")

	StartLoopApp(telnet)

	dummy.Send(NewDataMessage([]byte{255, 253, 0, 255, 253, 1}))              // DO BINARY & ECHO
	dummy.Send(NewDataMessage([]byte{255, 253, 3, 255, 251, 0, 255, 251, 3})) // DO SUPPRESS GO-AHEAD, WILL BIN & SGA

	dummy.Send(NewDataMessageFromString("Test"))
	o, ok = dummy.Recv()
	assert.Equal(t, true, ok, "No dummy receive error")

	if "Test" != o.(DataMessage).String() {
		t.Errorf("Result incorrect, expected 'Test', got '%s'", o)
	}

	// Make sure meaningless messages are processed okay
	n := NewConnectionMessage{}
	dummy.Send(n)
}

func TestTelnetReplacement(t *testing.T) {
	t.Parallel()

	table := [][][]byte{
		{{0, 1, 2, 3, 4, 5}, {0, 1, 2, 3, 4, 5}},
		{{}, {}},
		{{10, 0, 10, 0, 10}, {10, 13, 0, 10, 13, 0, 10, 13}},
		{{10, 0, 10, 255, 0, 10}, {10, 13, 0, 10, 13, 255, 255, 0, 10, 13}},
	}

	chars := []byte{10, 255}
	replace := [][]byte{{10, 13}, {255, 255}}

	for i := range table {
		assert.Equal(t, table[i][1], telnetReplaceBytes(table[i][0], chars, replace), "Replace successful")
	}
}
