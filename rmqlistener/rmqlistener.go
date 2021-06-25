package rmqlistener

import (
	"bytes"
	"encoding/json"
	"fmt"

	logger "github.com/dm1trypon/easy-logger"
	"github.com/streadway/amqp"
)

// Create <*RMQListener> - constructor of module RMQListener
//
// Returns:
// 	1. <*RMQListener>
// - pointer of object
func (r *RMQListener) Create(exchange, queue string, config Config, conn *amqp.Connection) *RMQListener {
	r = &RMQListener{
		lc:           "RMQ_LISTENER",
		lMask:        fmt.Sprint("[", exchange, ":", queue, "] "),
		conn:         conn,
		exchange:     exchange,
		queue:        queue,
		amqpDelivery: make(<-chan amqp.Delivery),
		amqpQueue:    amqp.Queue{},
		amqpChannel:  nil,
		msgChan:      make(chan string),
		config:       config,
	}

	return r
}

// GetConfig <*RMQListener> - getting settings of listener
//
// Returns:
// 	1. <Config>
// - settings of listener
func (r *RMQListener) GetConfig() Config {
	return r.config
}

// Subscribe <*RMQListener> - subscribing on events
//
// Args:
// 	1. consume <bool>
// - enable consuming
func (r *RMQListener) Subscribe(consume bool) {
	logger.InfoJ(r.lc, fmt.Sprint(r.lMask, "Subscribing"))

	var err error

	if !r.hasConnection() {
		logger.ErrorJ(r.lc, fmt.Sprint(r.lMask, "Subscribing is failed, RMQ connection is closed"))
		return
	}

	r.amqpChannel, err = r.conn.Channel()
	if err != nil {
		logger.ErrorJ(r.lc, fmt.Sprint(r.lMask, "Could not open RMQ channel: ", err.Error()))
		return
	}
	defer r.amqpChannel.Close()

	if err := r.amqpChannel.ExchangeDeclare(
		r.exchange,
		"direct",
		true,
		false,
		false,
		false,
		nil); err != nil {
		logger.ErrorJ(r.lc, fmt.Sprint(r.lMask, "Can not declare exchange: ", err.Error()))
		return
	}

	if !consume {
		return
	}

	r.amqpQueue, err = r.amqpChannel.QueueDeclare(r.queue, true, false, false, false, nil)
	if err != nil {
		logger.ErrorJ(r.lc, fmt.Sprint(r.lMask, "Error queue declaring: ", err.Error()))
		return
	}

	if err := r.amqpChannel.QueueBind(r.queue, "", r.exchange, false, nil); err != nil {
		logger.ErrorJ(r.lc, fmt.Sprint(r.lMask, "Error binding the queue: ", err.Error()))
		return
	}

	r.amqpDelivery, err = r.amqpChannel.Consume(r.queue, "", false, false, false, false, nil)
	if err != nil {
		logger.ErrorJ(r.lc, fmt.Sprint(r.lMask, "Error consuming the queue: ", err.Error()))
		return
	}

	r.consumer()
}

// GetChNextMsg <*RMQListener> - gets channel of incoming messages
//
// Returns:
// 	1. <<-chan string>
// - receive message channel
func (r *RMQListener) GetChNextMsg() <-chan string {
	return r.msgChan
}

// Publish <*RMQListener> - sends a Publishing from the client to an exchange on the server
//
// Args:
// 	1. body <[]byte>
// - body of message
// 	2. contentType <string>
// - content type of body
//
// Returns:
// 	1. <bool>
// - completion status
func (r *RMQListener) Publish(body []byte, contentType string) bool {
	if !r.hasConnection() {
		logger.ErrorJ(r.lc, fmt.Sprint(r.lMask, "Publishing is failed, RMQ connection is closed"))
		return false
	}

	pubData := amqp.Publishing{
		ContentType: contentType,
		Body:        body,
	}

	logger.DebugJ(r.lc, fmt.Sprint(r.lMask, "Publishing message: ", string(body)))

	amqpChannel, err := r.conn.Channel()
	if err != nil {
		logger.ErrorJ(r.lc, fmt.Sprint(r.lMask, "Could not open RMQ channel: ", err.Error()))
		return false
	}
	defer amqpChannel.Close()

	if err := amqpChannel.Publish(r.exchange, "", false, false, pubData); err != nil {
		logger.ErrorJ(r.lc, fmt.Sprint(r.lMask, "An error occurred while posting: ", err.Error()))
		return false
	}

	return true
}

// consumer <*RMQListener> - handler of RMQ events
func (r *RMQListener) consumer() {
	for {
		select {
		case msg, ok := <-r.amqpDelivery:
			if !ok {
				logger.ErrorJ(r.lc, fmt.Sprint(r.lMask, "Delivery channel is closed"))
				return
			}

			var body []byte

			compactMsg := new(bytes.Buffer)
			err := json.Compact(compactMsg, msg.Body)
			if err != nil {
				body = msg.Body
			} else {
				body = compactMsg.Bytes()
			}

			logger.DebugJ(r.lc, fmt.Sprint(r.lMask, "Imcoming message: ", string(body)))
			msg.Ack(true)
			r.msgChan <- string(body)
		}
	}
}

// hasConnection <*RMQListener> - checks RMQ connection state
//
// Returns:
// 	1. <bool>
// - has connection status
func (r *RMQListener) hasConnection() bool {
	return r.conn != nil || !r.conn.IsClosed()
}
