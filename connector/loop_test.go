package connector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoopDial(t *testing.T) {
	t.Parallel()

	dummy, err := NewDummyConnection("0")
	assert.Equal(t, nil, err, "No dummy connection error")

	StartLoopApp(dummy)

	dummy.Send(NewDataMessageFromString("Test"))
	o, ok := dummy.Recv()
	assert.Equal(t, true, ok, "No dummy receive error")

	if "Test" != o.(DataMessage).String() {
		t.Errorf("Result incorrect, expected 'Test', got '%s'", o)
	}

	// Make sure meaningless messages are processed okay
	n := NewConnectionMessage{}
	dummy.Send(n)
}
