package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jaanusjaeger/json-validation-service/internal/conf"
	"github.com/jaanusjaeger/json-validation-service/internal/server"
)

func main() {
	confFile := flag.String("conf", "conf.json", "The JSON configuration file")
	flag.Parse()

	conf, err := conf.LoadJSON(*confFile)
	if err != nil {
		log.Println("ERROR: loading conf:", err)
		os.Exit(1)
	}

	signalc := make(chan os.Signal, 1)
	defer close(signalc)
	signal.Notify(signalc, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signalc)

	srv := server.New(conf.Server)
	srvc := make(chan error, 1)
	go func() { srvc <- srv.ListenAndServe() }()

	log.Println("Server started at address", conf.Server.Addr)

	select {
	case err := <-srvc:
		log.Println("ERROR: server error:", err)
		os.Exit(1)
	case sig := <-signalc:
		signal.Stop(signalc)
		log.Printf("INFO: received signal %s, terminating\n", sig)
		if err := srv.Shutdown(10 * time.Second); err != nil {
			log.Println("ERROR: server shutdown error:", err)
		}
	}
}
