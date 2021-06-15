package rmqconnector

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"

	logger "github.com/dm1trypon/easy-logger"
	"github.com/dm1trypon/rmq-mdl/rmqlistener"
	"github.com/streadway/amqp"
	"vcs.bingo-boom.ru/offline/application/v1/event-v1-to-v3/servicedata"
)

const (
	ProtoInsecure = "amqp"
	ProtoSecure   = "amqps"
)

func (r *RMQConnector) Create() *RMQConnector {
	r = &RMQConnector{
		lc:            "RMQ_CONNECTOR",
		conn:          nil,
		internalErr:   nil,
		chStopService: make(chan bool, 1),
		mutex:         &sync.Mutex{},
		config: Config{
			Username: "user",
			Password: "password",
			Host:     "localhost",
			Port:     15672,
			TLS:      false,
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

func (r *RMQConnector) listenerHandler() {
	for kind, listener := range r.listeners {
		go func(kind string, listener *rmqlistener.RMQListener) {
			for {
				msg, ok := <-listener.GetMsgChan()
				if !ok {
					break
				}

				logger.DebugJ(r.lc, fmt.Sprint("[", kind, "] RECV: ", msg))
			}
		}(kind, listener)
	}
}

func (r *RMQConnector) setListeners() {
	r.listeners = map[string]*rmqlistener.RMQListener{}

	for _, event := range r.config.Events {
		r.listeners[event.Kind] = new(rmqlistener.RMQListener).Create(event.Exchange, event.Queue, r.conn)
	}
}

/*
GetChStopService <RMQConnector> - getting the service stop event channel
	Returns <<-chan bool>:
		1. event's channel
*/
func (r *RMQConnector) GetChStopService() <-chan bool {
	return r.chStopService
}

func (r *RMQConnector) SetConfig(config Config) {
	r.config = config
}

func (r *RMQConnector) RunOnce() bool {
	rmqURL := fmt.Sprint("://", r.config.Username, ":", r.config.Password, "@", r.config.Host, ":", r.config.Port)

	var err error

	if r.config.TLS {
		cfg := r.setTLS()
		if cfg == nil {
			return false
		}

		rmqURL = fmt.Sprint(ProtoSecure, rmqURL)
		r.conn, err = amqp.DialTLS(rmqURL, cfg)
		if err != nil {
			logger.ErrorJ(r.lc, fmt.Sprint("Failed connect to RMQ with TLS: ", err.Error()))
			return false
		}

		return true
	}

	rmqURL = fmt.Sprint(ProtoInsecure, rmqURL)

	r.conn, err = amqp.Dial(rmqURL)
	if err != nil {
		logger.ErrorJ(r.lc, fmt.Sprint("Failed connect to RMQ: ", err.Error()))
		return false
	}

	logger.InfoJ(r.lc, "Connected")

	go r.errorHandler()

	r.setListeners()

	for _, listener := range r.listeners {
		go listener.Subscribe()
	}

	r.listenerHandler()

	return true
}

func (r *RMQConnector) Run() {
	for {
		if !r.RunOnce() {
			time.Sleep(time.Second)
			continue
		}

		break
	}
}

func (r *RMQConnector) errorHandler() {
	r.internalErr = make(chan *amqp.Error)
	notify := r.conn.NotifyClose(r.internalErr)
	err := <-notify

	if err == nil {
		return
	}

	logger.ErrorJ(r.lc, fmt.Sprint("An occured error in RMQ connecting, description: [CODE][",
		err.Code, "] [REASON][", err.Reason, "]"))

	if !r.conn.IsClosed() {
		r.conn.Close()
		r.conn = nil
	}

	r.Run()
}

// setTLS - setting up secure connection to RMQ
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

// makeCerts - creates certificates and key by env value.
// Returns <bool>. True - is ok.
func (r *RMQConnector) makeCerts() bool {
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

// configuring - configures a TLS connection using certificates and keys located by path.
// Returns <bool>. True - is ok.
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
