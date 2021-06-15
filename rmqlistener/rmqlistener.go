package rmqlistener

import (
	"fmt"

	logger "github.com/dm1trypon/easy-logger"
	"github.com/streadway/amqp"
)

func (r *RMQListener) Create(exchange, queue string, conn *amqp.Connection) *RMQListener {
	r = &RMQListener{
		lc:           "RMQ_LISTENER",
		lMask:        fmt.Sprint("[", exchange, ":", queue, "] "),
		conn:         conn,
		internalErr:  nil,
		exchange:     exchange,
		queue:        queue,
		amqpDelivery: make(<-chan amqp.Delivery),
		amqpQueue:    amqp.Queue{},
		msgChan:      make(chan string),
	}

	return r
}

func (r *RMQListener) Subscribe() {
	logger.InfoJ(r.lc, fmt.Sprint(r.lMask, "Subscribing"))

	amqpChannel, err := r.conn.Channel()
	if err != nil {
		logger.ErrorJ(r.lc, fmt.Sprint(r.lMask, "Could not open RMQ channel: ", err.Error()))
		return
	}
	defer amqpChannel.Close()

	if err := amqpChannel.ExchangeDeclare(
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

	r.amqpQueue, err = amqpChannel.QueueDeclare(r.queue, true, false, false, false, nil)
	if err != nil {
		logger.ErrorJ(r.lc, fmt.Sprint(r.lMask, "ErrorJ queue declaring: ", err.Error()))
		return
	}

	if err := amqpChannel.QueueBind(r.queue, "", r.exchange, false, nil); err != nil {
		logger.ErrorJ(r.lc, fmt.Sprint(r.lMask, "ErrorJ binding the queue: ", err.Error()))
		return
	}

	r.amqpDelivery, err = amqpChannel.Consume(r.queue, "", false, false, false, false, nil)
	if err != nil {
		logger.ErrorJ(r.lc, fmt.Sprint(r.lMask, "ErrorJ consuming the queue: ", err.Error()))
		return
	}

	r.consumer()
}

func (r *RMQListener) GetMsgChan() chan string {
	return r.msgChan
}

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
