package connector

type Connector interface {
	Id() string
	Control() chan message
	Notify() chan message
}

type Connection interface {
	Id() string
	FromConn() chan message
	ToConn() chan message
}

type App interface {
	Id() string
}
