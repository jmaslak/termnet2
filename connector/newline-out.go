package connector

type newlineOutFilter struct {
	id                string
	inboundConnection Connection
	fromClient        chan message
	toClient          chan message
	lastChar          byte
}

func (filter newlineOutFilter) Id() string             { return filter.id }
func (filter newlineOutFilter) FromConn() chan message { return filter.fromClient }
func (filter newlineOutFilter) ToConn() chan message   { return filter.toClient }

func NewNewlineOutFilter(conn Connection) (newlineOutFilter, error) {
	filter := newlineOutFilter{}

	filter.inboundConnection = conn
	filter.id = conn.Id() + "-(newline)"
	filter.fillDefaults()

	go filter.doFilter()

	return filter, nil
}

func (filter *newlineOutFilter) fillDefaults() {
	filter.fromClient = make(chan message)
	filter.toClient = make(chan message)
}

func (filter *newlineOutFilter) doFilter() {
	defer close(filter.inboundConnection.ToConn())
	defer close(filter.fromClient)

	for {
		select {
		case m, ok := <-filter.inboundConnection.FromConn():
			if !ok {
				return
			}
			filter.processFromInboundConnection(m)
		case m, ok := <-filter.toClient:
			if !ok {
				return
			}
			filter.processToClient(m)
		}
	}
}

func (filter *newlineOutFilter) processToClient(m message) {
	// Process traffic needing to go out to the inboundConnection
	// (potentially).

	filter.inboundConnection.ToConn() <- m
}

func (filter *newlineOutFilter) processFromInboundConnection(m message) {
	// Process traffic coming from the inboundConnection needing to
	// go out the "fromClient" channel potentially
	if m.Type() != MTDataMessage {
		filter.fromClient <- m
		return
	}

	// We know we have a data message.
	msg := m.(DataMessage)
	out := make([]byte, 0)

	for i := range msg.Data {
		if msg.Data[i] == 10 {
			if filter.lastChar != 13 {
				out = append(out, 10, 13)
			}
		} else if msg.Data[i] == 13 {
			if filter.lastChar != 10 {
				out = append(out, 10, 13)
			}
		} else {
			out = append(out, msg.Data[i])
		}
		filter.lastChar = msg.Data[i]
	}

	if len(out) > 0 {
		msg := NewDataMessage(out)
		filter.fromClient <- msg
	}
}
