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
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	_ "go.uber.org/automaxprocs"
)

/*
	1. Get configuration from env variables
	2. Sort out things with https/http
*/

func main() {
	logger, err := initLogger()
	if err != nil {
		panic(err)
	}

	logger.Infow("startup", "GOMAXPROCS", runtime.GOMAXPROCS(0))

	cfg := Config{}
	help, err := conf.ParseOSArgs("CBREAKER", &cfg)
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			fmt.Println(help)
		}
		//Panic or smth
		//return fmt.Errorf("parsing config: %w", err)
	}

	proxyHandler := handlers.New(cfg.Settings.LongPooling, cfg.Settings.MaxFailedTries, cfg.Settings.OpenStateExpiry, logger)
	mux := http.NewServeMux()
	mux.HandleFunc("/", proxyHandler.PassThrough)

	go func() {
		if err := http.ListenAndServe(":8085", mux); err != nil {
			fmt.Println(err)
		}
	}()

	shutdown := make(chan os.Signal)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown
}

func initLogger() (*zap.SugaredLogger, error) {
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{"stdout"}
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.DisableStacktrace = true
	config.InitialFields = map[string]interface{}{
		"service": "circuit-breaker",
	}

	log, err := config.Build()
	if err != nil {
		return nil, err
	}

	return log.Sugar(), nil
}
