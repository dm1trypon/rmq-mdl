package rmqconnector

import (
	"sync"

	"github.com/dm1trypon/rmq-mdl/rmqlistener"
	"github.com/streadway/amqp"
)

type RMQConnector struct {
	lc            string
	conn          *amqp.Connection
	internalErr   chan *amqp.Error
	chStopService chan bool
	mutex         *sync.Mutex
	config        Config
	listeners     map[string]*rmqlistener.RMQListener
}

type Config struct {
	Username string
	Password string
	Host     string
	Port     uint16
	TLS      bool
	Certs    Certs
	Events   []Event
}

type Event struct {
	Kind     string
	Exchange string
	Queue    string
}

type Certs struct {
	InsecureSkipVerify bool
	Srcs               Srcs
	Paths              Paths
}

type Srcs struct {
	CA      []byte
	SrvCert []byte
	SrvKey  []byte
}

type Paths struct {
	CA      string
	SrvCert string
	SrvKey  string
}
