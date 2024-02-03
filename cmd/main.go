package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/ardanlabs/conf"
	"github.com/dnsx2k/circuit-breaker/cmd/handlers"
	_ "go.uber.org/automaxprocs"
	"log/slog"
)

/*
	1. Get configuration from env variables
	2. Sort out things with https/http
*/

func main() {
	slog.Info("startup", "GOMAXPROCS", runtime.GOMAXPROCS(0))

	cfg := Config{}
	help, err := conf.ParseOSArgs("CBREAKER", &cfg)
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			fmt.Println(help)
		} else {
			panic(err)
		}
	}

	proxyHandler := handlers.New(cfg.Settings.LongPooling, cfg.Settings.MaxFailedTries, cfg.Settings.OpenStateExpiry)
	mux := http.NewServeMux()
	mux.HandleFunc("/", proxyHandler.PassThrough)

	go func() {
		if err := http.ListenAndServe(":8085", mux); err != nil {
			panic(err)
		}
	}()

	shutdown := make(chan os.Signal)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown
}
