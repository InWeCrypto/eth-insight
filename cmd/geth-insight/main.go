package main

import (
	"flag"

	"github.com/dynamicgo/aliyunlog"
	"github.com/dynamicgo/config"
	"github.com/dynamicgo/slf4go"
	insight "github.com/inwecrypto/eth-insight"
	_ "github.com/lib/pq"
)

var logger = slf4go.Get("geth-indexer")
var configpath = flag.String("conf", "./geth-insight.json", "geth insight config file path")

func main() {

	flag.Parse()

	conf, err := config.NewFromFile(*configpath)

	if err != nil {
		logger.ErrorF("load neo config err , %s", err)
		return
	}

	factory, err := aliyunlog.NewAliyunBackend(conf)

	if err != nil {
		logger.ErrorF("create aliyun log backend err , %s", err)
		return
	}

	slf4go.Backend(factory)

	server, err := insight.NewServer(conf)

	if err != nil {
		logger.ErrorF("load neo config err , %s", err)
		return
	}

	server.Run()
}
