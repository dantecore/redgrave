package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/welkin/redgrave/internal/checker"
)

type PingRequest struct {
	URL     string `json:"url"`
	Timeout string `json:"timeout"`
}

type PingResponse struct {
	Success    bool   `json:"success"`
	StatusCode int    `json:"status_code"`
	LatencyMs  int64  `json:"latency_ms"`
	Error      string `json:"error,omitempty"`
}

func HandlePing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if err := validateURL(req.URL); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	timeout := 10 * time.Second
	if req.Timeout != "" {
		d, err := time.ParseDuration(req.Timeout)
		if err != nil {
			http.Error(w, "invalid timeout", http.StatusBadRequest)
			return
		}
		timeout = d
		if timeout <= 0 || timeout > 30*time.Second {
			http.Error(w, "timeout must be between 0 and 30 seconds", http.StatusBadRequest)
			return
		}
	}

	result := checker.Ping(req.URL, timeout)

	resp := PingResponse{
		Success:    result.Success,
		StatusCode: result.StatusCode,
		LatencyMs:  result.Latency.Milliseconds(),
		Error:      result.Error,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

}

// validateURL checks that rawURL is a valid http/https URL with a non-empty host.
func validateURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}

	switch parsed.Scheme {
	case "http", "https":
	default:
		return errors.New("only http and https schemes are allowed")
	}

	if parsed.Host == "" {
		return errors.New("url must include a host")
	}

	return nil
}
