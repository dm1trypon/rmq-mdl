package rmqconnector

import (
	"time"

	"github.com/dm1trypon/rmq-mdl/rmqlistener"
	"github.com/streadway/amqp"
)

// RMQConnector - main data of RMQConnector module
type RMQConnector struct {
	lc             string                              // logging category
	conn           *amqp.Connection                    // RMQ connection
	chErr          chan *amqp.Error                    // channel of RMQ connection error
	chNextMsg      chan Msg                            // channel of RMQ next message
	config         Config                              // RMQ settings
	listeners      map[string]*rmqlistener.RMQListener // list of RMQ listeners
	chConnected    chan bool                           // channel for connect events
	chDisconnected chan bool                           // channel for disconnect events
}

// Config - RMQ settings
type Config struct {
	Username             string        // username
	Password             string        // password
	ReconnectionInterval time.Duration // interval for reconnect
	Host                 string        // host
	Port                 uint16        // port
	TLS                  bool          // using TLS
	Certs                Certs         // data of Certificates, needed for TLS connection
	Events               []Event       // list of exchanges and queues in accordance with business logic
}

// Event - data for RMQ listener
type Event struct {
	Consuming bool   // enable or disable consuming
	Kind      string // kind of business logic
	Exchange  string // exchange
	Queue     string // queue
}

// Certs - data of Certificates, needed for TLS connection
type Certs struct {
	// InsecureSkipVerify controls whether a client verifies the server's
	// certificate chain and host name. If InsecureSkipVerify is true, crypto/tls
	// accepts any certificate presented by the server and any host name in that
	// certificate. In this mode, TLS is susceptible to machine-in-the-middle
	// attacks unless custom verification is used. This should be used only for
	// testing or in combination with VerifyConnection or VerifyPeerCertificate.
	InsecureSkipVerify bool
	Srcs               Srcs  // sourses data of certs
	Paths              Paths // data of certs paths
}

// Srcs - sourses data of certs
type Srcs struct {
	CA      []byte // root certificate
	SrvCert []byte // public certificate
	SrvKey  []byte // public key
}

// Paths - data of certs paths
type Paths struct {
	CA      string // root certificate
	SrvCert string // public certificate
	SrvKey  string // public key
}

// Msg - data that contains kind of buisnes logic and body of incoming message
type Msg struct {
	Kind string // kind of buisnes logic
	Body string // body of incoming message
}
