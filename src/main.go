package main

import (
	"flag"
	"os"

	"github.com/kafkaesque-io/pubsub-function/src/broker"
	"github.com/kafkaesque-io/pubsub-function/src/route"
	"github.com/kafkaesque-io/pubsub-function/src/util"
	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"
)

var mode = util.AssignString(os.Getenv("ProcessMode"), *flag.String("mode", "hybrid", "server running mode"))

func main() {
	exit := make(chan bool)
	util.Init()

	flag.Parse()
	log.Warnf("start server mode %s", mode)
	if !util.IsValidMode(&mode) {
		log.Panic("Unsupported server mode")
	}

	if util.IsBrokerRequired(&mode) {
		broker.Init()
	}

	if util.IsHTTPRouterRequired(&mode) {
		route.Init()

		c := cors.New(cors.Options{
			AllowedOrigins:   []string{"http://localhost:8085", "http://localhost:8080"},
			AllowCredentials: true,
			AllowedHeaders:   []string{"Authorization", "PulsarTopicUrl"},
		})

		router := route.NewRouter(&mode)

		handler := c.Handler(router)
		config := util.GetConfig()
		port := util.AssignString(config.PORT, "8081")
		certFile := util.GetConfig().CertFile
		keyFile := util.GetConfig().KeyFile
		log.Fatal(util.ListenAndServeTLS(":"+port, certFile, keyFile, handler))
	}

	select {
	case <-exit:
		os.Exit(2)
	}
}
