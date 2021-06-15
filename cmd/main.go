package main

import (
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
		Username: "guest",
		Password: "guest",
		Host:     "127.0.0.1",
		Port:     5672,
		TLS:      false,
		Certs:    rmqconnector.Certs{},
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

	<-rmqConn.GetChStopService()

	logger.InfoJ(LC, "STOPING SERVICE")
}
