package main // import "go.uber.org/sally"

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"go.uber.org/zap"
)

func main() {
	yml := flag.String("yml", "sally.yaml", "yaml file to read config from")
	port := flag.Int("port", 8080, "port to listen and serve on")
	flag.Parse()

	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to set up logger: %v", err)
		os.Exit(1)
	}

	logger.Debug("Parsing configuration", zap.String("path", *yml))

	config, err := Parse(*yml)
	if err != nil {
		logger.Fatal("Failed to parse configuration", zap.String("path", *yml), zap.Error(err))
	}

	logger.Info("Parsed configuration", zap.Object("config", config))
	handler := CreateHandler(config)

	addr := fmt.Sprintf(":%d", *port)
	logger.Info("Starting HTTP server", zap.String("addr", addr))

	if err := http.ListenAndServe(addr, handler); err != nil {
		logger.Fatal("Server stopped", zap.Error(err))
	}
}
