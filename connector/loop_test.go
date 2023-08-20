package connector

import "testing"

func TestLoopDial(t *testing.T) {
	t.Parallel()
	loop := NewLoop()

	in := make(chan message)
	out := make(chan message)

	loop.Dial(in, out)
	in <- NewTextMessage("Test")
	o := <-out

	if "Test" != o.(TextMessage).Text {
		t.Errorf("Result incorrect, expected 'Test', got '%s'", o)
	}

	// Make sure meaningless messages are processed okay
	n := NewConnectionMessage{}
	in <- n
}
