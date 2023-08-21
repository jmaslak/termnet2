package connector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewlineOutFilter(t *testing.T) {
	t.Parallel()

	dummy, err := NewDummyConnection("0")
	assert.Equal(t, nil, err, "No dummy connection error")

	filter, err := NewNewlineOutFilter(dummy)
	assert.Equal(t, nil, err, "No newline filter error")
	assert.Equal(t, dummy.Id()+"-(newline)", filter.Id(), "NewlineOut is proper")

	StartLoopApp(filter)

	dummy.Send(NewDataMessageFromString("Test\nTest\n\rFoo\rBar\n"))
	o, ok := dummy.Recv()
	assert.Equal(t, true, ok, "No dummy receive error")
	assert.Equal(t, "Test\n\rTest\n\rFoo\n\rBar\n\r", o.(DataMessage).String(), "Data is proper")

	// Make sure meaningless messages are processed okay
	n := NewConnectionMessage{}
	dummy.Send(n)
}
