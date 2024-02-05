package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/dnsx2k/circuit-breaker/cmd/handlers"
	"github.com/sethvargo/go-envconfig"
	_ "go.uber.org/automaxprocs"
	"log/slog"
)

func main() {
	slog.Info("startup", "GOMAXPROCS", runtime.GOMAXPROCS(0))

	cfg := Config{}
	if err := envconfig.Process(context.Background(), &cfg); err != nil {
		panic(err)
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
