package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kluwer/mtr-tool/internal/mtr"
	"github.com/rs/zerolog/log"
)

type MTRResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func HandleMTR(w http.ResponseWriter, r *http.Request) {
	// Extract and validate parameters
	hostname := r.URL.Query().Get("hostname")
	if hostname == "" {
		respondWithError(w, http.StatusBadRequest, "hostname parameter is required")
		return
	}

	// Validate hostname format
	if strings.ContainsAny(hostname, ";&|") {
		respondWithError(w, http.StatusBadRequest, "invalid hostname format")
		return
	}

	count := 20 // default value
	if countStr := r.URL.Query().Get("count"); countStr != "" {
		var err error
		count, err = strconv.Atoi(countStr)
		if err != nil || count <= 0 {
			respondWithError(w, http.StatusBadRequest, "invalid count parameter")
			return
		}
		if count > 100 {
			respondWithError(w, http.StatusBadRequest, "count cannot exceed 100")
			return
		}
	}

	report := false // default value
	if reportStr := r.URL.Query().Get("report"); reportStr != "" {
		var err error
		report, err = strconv.ParseBool(reportStr)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "invalid report parameter")
			return
		}
	}

	// Create MTR configuration
	cfg := mtr.Config{
		Hostname: hostname,
		Count:    count,
		Report:   report,
	}

	// Respond immediately that the request is being processed
	response := MTRResponse{
		Status:  "accepted",
		Message: fmt.Sprintf("MTR trace to %s started (count=%d, report=%v)", hostname, count, report),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	// Run MTR command asynchronously
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		log.Info().
			Str("hostname", hostname).
			Int("count", count).
			Bool("report", report).
			Msg("Starting MTR trace")

		result, err := mtr.Run(ctx, cfg)
		if err != nil {
			log.Error().Err(err).Msg("MTR trace failed")
			fmt.Printf("\nMTR trace to %s failed: %v\n", hostname, err)
			return
		}

		// Print the result to console
		fmt.Printf("\nMTR trace to %s completed:\n%s\n", hostname, result.Output)
	}()
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	response := MTRResponse{
		Status:  "error",
		Message: message,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(response)
}
