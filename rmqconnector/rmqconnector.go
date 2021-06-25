package rmqconnector

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	logger "github.com/dm1trypon/easy-logger"
	"github.com/dm1trypon/rmq-mdl/rmqlistener"
	"github.com/streadway/amqp"
	"vcs.bingo-boom.ru/offline/application/v1/event-v1-to-v3/servicedata"
)

// Protocols
const (
	ProtoInsecure = "amqp"
	ProtoSecure   = "amqps"
)

// Create <*RMQConnector> - constructor of module RMQConnector
//
// Returns:
// 	1. <*RMQConnector>
// - pointer of object
func (r *RMQConnector) Create() *RMQConnector {
	r = &RMQConnector{
		lc:             "RMQ_CONNECTOR",
		conn:           nil,
		chErr:          nil,
		chNextMsg:      make(chan Msg),
		chConnected:    make(chan bool),
		chDisconnected: make(chan bool),
		config: Config{
			Username:             "user",
			Password:             "password",
			Host:                 "localhost",
			Port:                 15672,
			TLS:                  false,
			ReconnectionInterval: time.Second,
			Certs: Certs{
				InsecureSkipVerify: true,
				Srcs: Srcs{
					CA:      []byte{},
					SrvCert: []byte{},
					SrvKey:  []byte{},
				},
				Paths: Paths{
					CA:      "rmq-ca-cert.pem",
					SrvCert: "rmq-server-cert.pem",
					SrvKey:  "rmq-server-key.pem",
				},
			},
			Events: []Event{},
		},
		listeners: map[string]*rmqlistener.RMQListener{},
	}

	return r
}

// GetChConnected <*RMQConnector> - gets channel of connection events
//
// Returns:
// 	1. <<-chan bool>
// - receive connection events channel
func (r *RMQConnector) GetChConnected() <-chan bool {
	return r.chConnected
}

// GetChDisconnected <*RMQConnector> - gets channel of disconnection events
//
// Returns:
// 	1. <<-chan bool>
// - receive disconnection events channel
func (r *RMQConnector) GetChDisconnected() <-chan bool {
	return r.chDisconnected
}

// Publish <*RMQConnector> - publish message by kind of buisnes logic
//
// Args:
//	1.body <[]byte>
// - body of message
// 	2. kind <string>
// - kind of buisnes logic
// 	3. contentType <string>
// - content type of body
//
// Returns:
// 	1. <bool>
// - completion status
func (r *RMQConnector) Publish(body []byte, kind, contentType string) bool {
	for key, listener := range r.listeners {
		if key == kind {
			return listener.Publish(body, contentType)
		}
	}

	return false
}

// listenerHandler <*RMQConnector> - handler on new meesages from RMQListener
func (r *RMQConnector) listenerHandler() {
	for kind, listener := range r.listeners {
		go r.nextMessage(kind, listener)
	}
}

// Create <*RMQConnector> - constructor of module RMQConnector
//
// Args:
// 	1. kind <string>
// - kind of buisnes logic
// 	2. listener <*rmqlistener.RMQListener>
// - listener of RMQ module
func (r *RMQConnector) nextMessage(kind string, listener *rmqlistener.RMQListener) {
	for {
		msg, ok := <-listener.GetChNextMsg()
		if !ok {
			break
		}

		r.chNextMsg <- Msg{
			Kind: kind,
			Body: msg,
		}
	}
}

// setListeners <*RMQConnector> - creates listeners for RMQ module
func (r *RMQConnector) setListeners() {
	r.listeners = map[string]*rmqlistener.RMQListener{}

	for _, event := range r.config.Events {
		config := rmqlistener.Config{
			Consuming: event.Consuming,
		}

		r.listeners[event.Kind] = new(rmqlistener.RMQListener).Create(event.Exchange, event.Queue, config, r.conn)
	}
}

// GetChNextMsg <*RMQConnector> - gets channel of incoming messages
//
// Returns:
// 	1. <<-chan Msg>
// - receive message channel
func (r *RMQConnector) GetChNextMsg() <-chan Msg {
	return r.chNextMsg
}

// SetConfig <*RMQConnector> - sets RMQ settings
//
// Args:
// 	1. config <Config>
// - settings object for RMQ module
func (r *RMQConnector) SetConfig(config Config) {
	r.config = config
}

// RunOnce <*RMQConnector> - runs the logic for connecting to RMQ once
//
// Returns:
// 	1. <bool>
// - completion status
func (r *RMQConnector) RunOnce() bool {
	rmqURL := fmt.Sprint("://", r.config.Username, ":", r.config.Password, "@", r.config.Host, ":", r.config.Port)

	var err error

	if r.config.TLS {
		cfg := r.setTLS()
		if cfg == nil {
			logger.WarningJ(r.lc, fmt.Sprint("TLS connection is not possible, unsecured connection is used"))
			r.config.TLS = false
		} else {
			rmqURL = fmt.Sprint(ProtoSecure, rmqURL)
			logger.DebugJ(r.lc, fmt.Sprint("Connecting to ", rmqURL))
			r.conn, err = amqp.DialTLS(rmqURL, cfg)
			if err != nil {
				logger.ErrorJ(r.lc, fmt.Sprint("Failed connect to RMQ with TLS: ", err.Error()))
				return false
			}

			logger.InfoJ(r.lc, "Connected")

			go r.listenerHandler()

			r.chConnected <- true
			return true
		}
	}

	rmqURL = fmt.Sprint(ProtoInsecure, rmqURL)
	logger.DebugJ(r.lc, fmt.Sprint("Connecting to ", rmqURL))
	r.conn, err = amqp.Dial(rmqURL)
	if err != nil {
		logger.ErrorJ(r.lc, fmt.Sprint("Failed connect to RMQ: ", err.Error()))
		return false
	}

	logger.InfoJ(r.lc, "Connected")

	go r.errorHandler()

	r.setListeners()

	for _, listener := range r.listeners {
		cfg := listener.GetConfig()

		go listener.Subscribe(cfg.Consuming)
	}

	go r.listenerHandler()

	r.chConnected <- true
	return true
}

// Run <*RMQConnector> - runs the logic for looping connections to the RMQ in case of errors
func (r *RMQConnector) Run() {
	for {
		if !r.RunOnce() {
			time.Sleep(r.config.ReconnectionInterval)
			continue
		}

		break
	}
}

// errorHandler <*RMQConnector> - handler for error connection to RMQ server
func (r *RMQConnector) errorHandler() {
	r.chErr = make(chan *amqp.Error)
	notify := r.conn.NotifyClose(r.chErr)
	err := <-notify

	if err == nil {
		return
	}

	logger.ErrorJ(r.lc, fmt.Sprint("An occured error in RMQ connecting, description: [CODE][",
		err.Code, "] [REASON][", err.Reason, "]"))

	if err := r.conn.Close(); err != nil {
		logger.ErrorJ(r.lc, fmt.Sprint("Closing connection error: ", err.Error()))
	}

	r.chDisconnected <- true
	r.Run()
}

// setTLS <*RMQConnector> - setting up secure connection to RMQ
//
// Returns:
// 	1. <*tls.Config>
// - pointer of TLS config object
func (r *RMQConnector) setTLS() *tls.Config {
	if len(r.config.Certs.Srcs.CA) > 0 &&
		len(r.config.Certs.Srcs.SrvCert) > 0 &&
		len(r.config.Certs.Srcs.SrvKey) > 0 {
		if !r.makeCerts() {
			return nil
		}
	}

	return r.configuring()
}

// makeCerts <*RMQConnector> - creates certificates and key by env value
//
// Returns:
// 	1. <bool>
// - completion status
func (r *RMQConnector) makeCerts() bool {
	if err := os.MkdirAll(filepath.Dir(r.config.Certs.Paths.CA), os.ModePerm); err != nil {
		logger.Error(r.lc, fmt.Sprint("Failed to create ssl directory: ", err.Error()))
		return false
	}

	var certs = map[string][]byte{
		r.config.Certs.Paths.CA:      r.config.Certs.Srcs.CA,
		r.config.Certs.Paths.SrvCert: r.config.Certs.Srcs.SrvCert,
		r.config.Certs.Paths.SrvKey:  r.config.Certs.Srcs.SrvKey,
	}

	for path, data := range certs {
		file, err := os.Create(path)
		if err != nil {
			logger.ErrorJ(r.lc, fmt.Sprint("Unable to create certificate ", path, ": ", err.Error()))
			return false
		}

		_, err = file.Write(data)
		if err != nil {
			logger.ErrorJ(r.lc, fmt.Sprint("Unable to write certificate ", path, ": ", err.Error()))
			return false
		}

		file.Close()

		logger.InfoJ(r.lc, fmt.Sprint("Certificate ", path, " created"))
	}

	return true
}

// configuring <*RMQConnector> - configures a TLS connection using certificates and keys located by path
//
// Returns:
// 	1. <bool>
// - completion status
func (r *RMQConnector) configuring() *tls.Config {
	cfg := &tls.Config{
		Rand:                        nil,
		Certificates:                []tls.Certificate{},
		NameToCertificate:           map[string]*tls.Certificate{},
		RootCAs:                     x509.NewCertPool(),
		NextProtos:                  []string{},
		ServerName:                  "",
		ClientAuth:                  0,
		ClientCAs:                   &x509.CertPool{},
		InsecureSkipVerify:          r.config.Certs.InsecureSkipVerify,
		CipherSuites:                []uint16{},
		PreferServerCipherSuites:    false,
		SessionTicketsDisabled:      false,
		SessionTicketKey:            [32]byte{},
		ClientSessionCache:          nil,
		MinVersion:                  0,
		MaxVersion:                  0,
		CurvePreferences:            []tls.CurveID{},
		DynamicRecordSizingDisabled: false,
		Renegotiation:               0,
		KeyLogWriter:                nil,
	}

	if ca, err := ioutil.ReadFile(servicedata.Config.RMQ.PathCACrt); err == nil {
		if ok := cfg.RootCAs.AppendCertsFromPEM(ca); !ok {
			logger.ErrorJ(r.lc, "Failed to append PEM")
			return nil
		}
	} else {
		logger.ErrorJ(r.lc, fmt.Sprint("Error reading certs: ", err.Error()))
		return nil
	}

	// Move the client cert and key to a location specific to your application
	// and load them here.
	if cert, err := tls.LoadX509KeyPair(servicedata.Config.RMQ.PathCrt, servicedata.Config.RMQ.PathKey); err == nil {
		cfg.Certificates = append(cfg.Certificates, cert)
	} else {
		logger.ErrorJ(r.lc, fmt.Sprint("LoadX509KeyPair reading error: ", err.Error()))
		return nil
	}

	return cfg
}
