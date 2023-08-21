package connector

import (
	"log"
	"reflect"
	"regexp"
)

type telnetFilter struct {
	id                string
	inboundConnection Connection
	fromClient        chan message
	toClient          chan message
	readBuffer        []byte // Store partial reads, such as data terminating in an IAC character
	writeBuffer       []byte // Used when we are still negotiating with client before sending
	charInterrupt     byte
	pendingDo         map[telnetOption]bool // Pending DO commands
	pendingWill       map[telnetOption]bool // Pending WILL commands
	optReceiveBinary  bool
	optSendBinary     bool
	regexpNewline     regexp.Regexp
}

const (
	telnetGoAhead byte = 249
	telnetWill    byte = 251
	telnetWont    byte = 252
	telnetDo      byte = 253
	telnetDont    byte = 254
	telnetIAC     byte = 255
)

type telnetOption byte

const (
	telnetOptBinary       telnetOption = iota
	telnetOptEcho                      // Implemented on server side only
	telnetOptReconnection              // Not supported, not RFC
	telnetOptSuppressGoAhead
	telnetOptTimingMark
)

func (opt telnetOption) Byte() byte {
	return byte(reflect.ValueOf(opt).Uint())
}

func (telnet telnetFilter) Id() string             { return telnet.id }
func (telnet telnetFilter) FromConn() chan message { return telnet.fromClient }
func (telnet telnetFilter) ToConn() chan message   { return telnet.toClient }

func NewTelnetFilter(conn Connection) (telnetFilter, error) {
	telnet := telnetFilter{}

	telnet.inboundConnection = conn
	telnet.id = conn.Id() + "-(telnet)"
	telnet.fillDefaults()

	go telnet.doFilter()

	return telnet, nil
}

func (telnet *telnetFilter) fillDefaults() {
	telnet.fromClient = make(chan message)
	telnet.toClient = make(chan message)
	telnet.pendingDo = make(map[telnetOption]bool)
	telnet.pendingWill = make(map[telnetOption]bool)
	telnet.charInterrupt = 3
	telnet.optReceiveBinary = false
	telnet.optSendBinary = false
	telnet.readBuffer = make([]byte, 0)
	telnet.writeBuffer = make([]byte, 0)
}

func (telnet *telnetFilter) doFilter() {
	defer close(telnet.inboundConnection.ToConn())
	defer close(telnet.fromClient)

	telnet.initNegotiate()

	for {
		select {
		case m, ok := <-telnet.inboundConnection.FromConn():
			if !ok {
				return
			}
			telnet.processFromInboundConnection(m)
		case m, ok := <-telnet.toClient:
			if !ok {
				return
			}
			telnet.processToClient(m)
		}
	}
}

func (telnet *telnetFilter) processFromInboundConnection(m message) {
	// Process traffic coming from the inboundConnection needing to
	// go out the "fromClient" channel potentially
	if m.Type() != MTDataMessage {
		telnet.fromClient <- m
		return
	}

	// We know we have a data message.
	msg := m.(DataMessage)
	b := append(telnet.readBuffer, msg.Data...)
	telnet.readBuffer = make([]byte, 0)

	telnet.processInboundData(b)
}

func (telnet *telnetFilter) processInboundData(b []byte) {
	// We've extracted the DataMessage and have raw bytes now
	l := len(b)
	if l == 0 {
		return
	}

	out := make([]byte, 0)
	skipNext := 0
	for i := range b {
		if skipNext > 0 {
			skipNext--
			continue
		} else if b[i] == 255 {
			if (l - 1) == i {
				// No character following!
				telnet.readBuffer = []byte{255}
				continue
			} else if b[i+1] == 255 {
				// Escaped escape charachter
				skipNext = 1
				out = append(out, 255)
				continue
			} else if b[i+1] == 249 {
				skipNext = 1
				// Go Ahead, which we just eat.
			} else if b[i+1] == 244 {
				// IP (Interrupt Process)
				skipNext = 1
				out = append(out, telnet.charInterrupt)
				continue
			} else if b[i+1] == 241 {
				// NOOP, so we do nothing
				skipNext = 1
				continue
			} else if b[i+1] < 251 {
				// We don't know what to do with it. So
				// just will eat it and puke anything
				// following it back to the downstream
				// as if it is text.
				log.Printf("Received invalid option code (%d)", b[i+1])
				skipNext = 1
				continue
			}

			// Now we know we're processing a command.
			if (l - 2) == i {
				// No option following the command.
				telnet.readBuffer = []byte{255, b[i+1]}
				skipNext = 1
				continue
			}

			// We know we have enough data to process the
			// command
			skipNext = 2

			switch b[i+1] {
			case 251:
				telnet.handleWill(telnetOption(b[i+2]))
			case 252:
				telnet.handleWont(telnetOption(b[i+2]))
			case 253:
				telnet.handleDo(telnetOption(b[i+2]))
			case 254:
				telnet.handleDont(telnetOption(b[i+2]))
			}
			continue
		} else if b[i] == 10 && !telnet.optReceiveBinary {
			// New Line
			out = append(out, 10, 13)
			continue
		} else if b[i] == 0 && !telnet.optReceiveBinary {
			// NUL
			continue
		} else {
			// Literally everything else!
			out = append(out, b[i])
		}
	}

	if len(out) > 0 {
		msg := NewDataMessage(out)
		telnet.fromClient <- msg
	}
}

func (telnet *telnetFilter) processToClient(m message) {
	// Process traffic needing to go out to the inboundConnection
	// (potentially).

	if m.Type() == MTDataMessage {
		if len(telnet.pendingWill) > 0 {
			telnet.writeBuffer = append(telnet.writeBuffer, m.(DataMessage).Data...)
			return
		}

		telnet.sendBytes(m.(DataMessage).Data)
	}

}

func (telnet *telnetFilter) sendBytes(b []byte) {
	var replaced []byte
	if !telnet.optSendBinary {
		replaced = telnetReplaceBytes(b, []byte{10, 255}, [][]byte{{10, 13}, {255, 255}})
	} else {
		replaced = telnetReplaceBytes(b, []byte{255}, [][]byte{{255, 255}})
	}

	m := NewDataMessage(replaced)
	telnet.inboundConnection.ToConn() <- m
}

func (telnet *telnetFilter) initNegotiate() {
	// Send initial negotiation, currently negotiating bidirectional
	// binary mode, no echo, and no go-ahead messages.
	telnet.pendingWill[telnetOptBinary] = true
	telnet.pendingWill[telnetOptEcho] = true
	telnet.pendingWill[telnetOptSuppressGoAhead] = true

	telnet.pendingDo[telnetOptBinary] = true
	telnet.pendingDo[telnetOptEcho] = false
	telnet.pendingDo[telnetOptSuppressGoAhead] = true

	telnet.sendWill(telnetOptBinary)
	telnet.sendWill(telnetOptEcho)
	telnet.sendWill(telnetOptSuppressGoAhead)

	telnet.sendDo(telnetOptBinary)
	// We don't send a don't echo because that's the default mode,
	// and default mode options don't warrant an acknowledgement
	telnet.sendDo(telnetOptSuppressGoAhead)
}

func (telnet *telnetFilter) sendWill(opt telnetOption) {
	will := []byte{telnetIAC, telnetWill, opt.Byte()}
	msg := NewDataMessage(will)
	telnet.inboundConnection.ToConn() <- msg
}

func (telnet *telnetFilter) sendWont(opt telnetOption) {
	wont := []byte{telnetIAC, telnetWont, opt.Byte()}
	msg := NewDataMessage(wont)
	telnet.inboundConnection.ToConn() <- msg
}

func (telnet *telnetFilter) sendDo(opt telnetOption) {
	do := []byte{telnetIAC, telnetDo, opt.Byte()}
	msg := NewDataMessage(do)
	telnet.inboundConnection.ToConn() <- msg
}

func (telnet *telnetFilter) sendDont(opt telnetOption) {
	dont := []byte{telnetIAC, telnetDont, opt.Byte()}
	msg := NewDataMessage(dont)
	telnet.inboundConnection.ToConn() <- msg
}

func telnetReplaceBytes(src []byte, c []byte, replace [][]byte) []byte {
	// Takes src, and anywhere one of the characters "c" occurs,
	// replaces that with the string in the associated index in
	// replace.
	b := make([]byte, 0, len(src))
	for i := range src {
		flag := false
		for j := range c {
			if src[i] == c[j] {
				b = append(b, replace[j]...)
				flag = true
				break
			}
		}
		if !flag {
			b = append(b, src[i])
		}
	}

	return b
}

func (telnet *telnetFilter) ackIfNeeded(opt telnetOption, response string) {
	// This will send a response of type "response" if that would
	// not be duplicating a previously sent response that wasn't
	// acked/nacked by the other end.  It, as a side effect, removes
	// the option from pendingDo or pendingWill response lists
	// depending on if the response is a Do/Dont or Will/Wont, so
	// that future packets of this type would get acked.

	// Send do/don't
	if response == "DO" {
		_, ok := telnet.pendingDo[opt]
		if ok {
			delete(telnet.pendingDo, opt)
		} else {
			telnet.sendDo(opt)
		}
		return
	} else if response == "DONT" {
		_, ok := telnet.pendingDo[opt]
		if ok {
			delete(telnet.pendingDo, opt)
		} else {
			telnet.sendDont(opt)
		}
		return
	}

	if response != "WILL" && response != "WONT" {
		log.Fatal("Unknown response type: " + response)
	}

	// So we should have a response to a will/won't, which might trigger us
	// sending data.
	_, ok := telnet.pendingWill[opt]
	if ok {
		delete(telnet.pendingWill, opt)
		if len(telnet.pendingWill) == 0 {
			if len(telnet.writeBuffer) > 0 {
				data := telnet.writeBuffer
				telnet.writeBuffer = make([]byte, 0)
				telnet.sendBytes(data)
			}
		}
		return
	}

	// We have a will/won't, but no pending ack, so we need to ack.
	if response == "WILL" {
		telnet.sendWill(opt)
	} else if response == "WONT" {
		telnet.sendWont(opt)
	}
}

func (telnet *telnetFilter) handleWill(opt telnetOption) {
	response := "DONT" // Default response
	if opt == telnetOptBinary {
		telnet.optReceiveBinary = true
		response = "DO"
	} else if opt == telnetOptEcho {
		// We never want to have the client do the echo!
		response = "DO"
	} else if opt == telnetOptSuppressGoAhead {
		response = "DO"
	}

	telnet.ackIfNeeded(opt, response)
}

func (telnet *telnetFilter) handleWont(opt telnetOption) {
	response := "DONT" // Default response
	if opt == telnetOptBinary {
		telnet.optReceiveBinary = false
	}

	telnet.ackIfNeeded(opt, response)
}

func (telnet *telnetFilter) handleDo(opt telnetOption) {
	response := "WONT" // Default response
	if opt == telnetOptBinary {
		telnet.optSendBinary = true
		response = "WILL"
	} else if opt == telnetOptEcho {
		response = "WILL"
	} else if opt == telnetOptSuppressGoAhead {
		// We don't actually send GoAheads anyhoow.
		response = "WILL"
	}

	telnet.ackIfNeeded(opt, response)
}

func (telnet *telnetFilter) handleDont(opt telnetOption) {
	response := "WONT" // Default response
	if opt == telnetOptBinary {
		telnet.optSendBinary = false
	}

	telnet.ackIfNeeded(opt, response)
}
