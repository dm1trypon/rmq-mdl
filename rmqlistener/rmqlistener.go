package rmqlistener

import (
	"fmt"

	logger "github.com/dm1trypon/easy-logger"
	"github.com/streadway/amqp"
)

// Create <*RMQListener> - constructor of module RMQListener
//
// Returns:
// 	1. <*RMQListener>
// - pointer of object
func (r *RMQListener) Create(exchange, queue string, conn *amqp.Connection) *RMQListener {
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
	}

	return r
}

// Subscribe <*RMQListener> - subscribing on events
func (r *RMQListener) Subscribe() {
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
	if !r.hasConnection() || r.amqpChannel == nil {
		logger.ErrorJ(r.lc, fmt.Sprint(r.lMask, "Publishing is failed, RMQ connection is closed"))
		return false
	}

	pubData := amqp.Publishing{
		ContentType: contentType,
		Body:        body,
	}

	logger.DebugJ(r.lc, fmt.Sprint(r.lMask, "Publishing message: ", string(body)))

	if err := r.amqpChannel.Publish(r.exchange, "", false, false, pubData); err != nil {
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

			logger.DebugJ(r.lc, fmt.Sprint(r.lMask, "Imcoming message: ", string(msg.Body)))
			r.msgChan <- string(msg.Body)
			msg.Ack(true)
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
