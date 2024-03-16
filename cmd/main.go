package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/dnsx2k/gofuse/cmd/handlers"
	"github.com/dnsx2k/gofuse/pkg/settings"
	_ "go.uber.org/automaxprocs"
)

func main() {
	slog.Info("startup", "GOMAXPROCS", runtime.GOMAXPROCS(0))

	configFilePath := os.Getenv("GOFUSE_CLIENTS_CONFIG")
	bytes, err := os.ReadFile(configFilePath)
	if err != nil {
		panic(err)
	}
	var cfg map[string]settings.ClientConfiguration
	if err = json.Unmarshal(bytes, &cfg); err != nil {
		panic(err)
	}

	proxyHandler := handlers.New(cfg)
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
