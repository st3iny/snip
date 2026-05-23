package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"snip.io/internal/cfg"
	"snip.io/internal/server"
)

const pidFilePath = "/var/tmp/snip.pid"

func main() {
	confPath := flag.String("config", "/etc/snip/config.toml", "Path to the config file")
	flag.Parse()

	pid := os.Getpid()
	log.Println("Snip running with pid", pid)
	err := os.WriteFile(pidFilePath, fmt.Append([]byte{}, pid), 0644)
	if err != nil {
		log.Fatal("Failed to write pid file:", err)
	}
	defer os.Remove(pidFilePath)

	conf, err := cfg.Parse(*confPath)
	if err != nil {
		log.Fatal("Failed to parse config:", err)
	}

	remainingArgs := flag.Args()
	if len(remainingArgs) > 0 && remainingArgs[0] == "validate" {
		log.Default().Println("Config is valid")
		return
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	reloadSigs := make(chan os.Signal, 1)
	signal.Notify(reloadSigs, syscall.SIGUSR1)

	confChannel := make(chan *cfg.Conf, 1)
	go func() {
		for {
			<-reloadSigs
			log.Println("Reloading config")

			conf, err := cfg.Parse(*confPath)
			if err != nil {
				log.Println("Keeping old config:", err)
				continue
			}

			confChannel <- conf
		}
	}()

	serverDone := make(chan any, 1)
	for ctx.Err() == nil {
		serverCtx, cancel := context.WithCancel(ctx)

		go func() {
			srv := server.New(conf)
			srv.Run(serverCtx)
			serverDone <- nil
		}()

		select {
		case <-ctx.Done():
			break
		case conf = <-confChannel:
			break
		}

		cancel()
		<-serverDone
	}

	log.Println("Server shutting down (waiting for pending connections)")
	server.WaitForConnections()
}
