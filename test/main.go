package main

import (
	"fmt"
	"time"

	logger "github.com/dm1trypon/easy-logger"
	"github.com/dm1trypon/rmq-mdl/rmqconnector"
)

// LC - logging's category
const LC = "MAIN"

func main() {
	logCfg := logger.Cfg{
		AppName: "RMQ_MDL",
		LogPath: "",
		Level:   0,
	}

	logger.SetConfig(logCfg)

	logger.InfoJ(LC, "STARTING SERVICE")

	rmqConn := new(rmqconnector.RMQConnector).Create()

	cfg := rmqconnector.Config{
		Username:             "guest",
		Password:             "guest",
		Host:                 "127.0.0.1",
		Port:                 5672,
		TLS:                  true,
		ReconnectionInterval: time.Duration(2 * time.Second),
		Certs:                rmqconnector.Certs{},
		Events: []rmqconnector.Event{
			{
				Kind:     "logic",
				Exchange: "test",
				Queue:    "test_queue",
			},
			{
				Kind:     "logic1",
				Exchange: "test1",
				Queue:    "test_queue1",
			},
			{
				Kind:     "logic2",
				Exchange: "test2",
				Queue:    "test_queue2",
			},
		},
	}

	rmqConn.SetConfig(cfg)
	go rmqConn.Run()
	<-rmqConn.GetChConnected()

	for {
		msg := <-rmqConn.GetChNextMsg()
		logger.DebugJ(LC, fmt.Sprint("[", msg.Kind, "] RECV: ", msg.Body))
	}
}
