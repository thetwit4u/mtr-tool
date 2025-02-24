package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kluwer/mtr-tool/internal/api"
	"github.com/kluwer/mtr-tool/internal/mtr"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Parse command line flags
	var (
		serverMode = flag.Bool("server", false, "Run in server mode")
		port       = flag.String("port", "8080", "Server port (only in server mode)")
		hostname   = flag.String("host", "", "Target hostname (only in CLI mode)")
		count      = flag.Int("count", 20, "Number of packets to send")
		report     = flag.Bool("report", false, "Enable report mode")
	)
	flag.Parse()

	if *serverMode {
		// Configure logging for server mode
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
		runServer(*port)
	} else {
		runCLI(*hostname, *count, *report)
	}
}

func runServer(port string) {
	// Create router and configure routes
	r := mux.NewRouter()
	r.HandleFunc("/mtr", api.HandleMTR).Methods("GET")

	// Configure server
	addr := "0.0.0.0:" + port
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Info().Msgf("Starting server on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed to start")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful shutdown
	log.Info().Msg("Server is shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exited properly")
}

func runCLI(hostname string, count int, report bool) {
	if hostname == "" {
		fmt.Println("Error: hostname is required")
		flag.Usage()
		os.Exit(1)
	}

	cfg := mtr.Config{
		Hostname: hostname,
		Count:    count,
		Report:   report,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := mtr.Run(ctx, cfg)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(result.Output)
}
