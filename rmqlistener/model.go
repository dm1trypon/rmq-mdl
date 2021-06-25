package rmqlistener

import "github.com/streadway/amqp"

// RMQListener - main data of RMQListener module
type RMQListener struct {
	lc           string               // logging category
	lMask        string               // logging mask
	conn         *amqp.Connection     // RMQ connection
	exchange     string               // exchange
	queue        string               // queue
	amqpDelivery <-chan amqp.Delivery // channel of RMQ delivery
	amqpQueue    amqp.Queue           // queue data
	amqpChannel  *amqp.Channel        // pointer of channel data
	msgChan      chan string          // channel of incoming messages
	config       Config               // settings of RMQ listener
}

// Config - settings of RMQ listener
type Config struct {
	Consuming bool // enable or disable consuming
}
