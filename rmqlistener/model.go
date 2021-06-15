package rmqlistener

import "github.com/streadway/amqp"

type RMQListener struct {
	lc           string
	lMask        string
	conn         *amqp.Connection
	internalErr  chan *amqp.Error
	exchange     string
	queue        string
	amqpDelivery <-chan amqp.Delivery
	amqpQueue    amqp.Queue
	msgChan      chan string
}
