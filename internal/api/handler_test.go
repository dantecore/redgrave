package api

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func doPing(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/ping", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	HandlePing(w, req)
	return w
}

// --- validation-only tests (no real outbound HTTP) ---

func TestHandlePing_WrongMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	HandlePing(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandlePing_InvalidJSON(t *testing.T) {
	w := doPing(t, "{")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid json") {
		t.Fatalf("expected 'invalid json', got %q", w.Body.String())
	}
}

func TestHandlePing_EmptyURL(t *testing.T) {
	w := doPing(t, `{"url":""}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandlePing_NonHTTPScheme(t *testing.T) {
	w := doPing(t, `{"url":"ftp://example.com"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "only http and https") {
		t.Fatalf("expected scheme error, got %q", w.Body.String())
	}
}

func TestHandlePing_NoHost(t *testing.T) {
	w := doPing(t, `{"url":"http://"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "host") {
		t.Fatalf("expected host error, got %q", w.Body.String())
	}
}

func TestHandlePing_InvalidTimeout(t *testing.T) {
	w := doPing(t, `{"url":"http://example.com","timeout":"abc"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid timeout") {
		t.Fatalf("expected 'invalid timeout', got %q", w.Body.String())
	}
}

func TestHandlePing_ZeroTimeout(t *testing.T) {
	w := doPing(t, `{"url":"http://example.com","timeout":"0s"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandlePing_NegativeTimeout(t *testing.T) {
	w := doPing(t, `{"url":"http://example.com","timeout":"-1s"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandlePing_TimeoutBelow1s(t *testing.T) {
	w := doPing(t, `{"url":"http://example.com","timeout":"500ms"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandlePing_TimeoutExceeds30s(t *testing.T) {
	w := doPing(t, `{"url":"http://example.com","timeout":"31s"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// --- integration tests (real outbound HTTP via httptest server) ---

func TestHandlePing_Success2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	body := `{"url":"` + srv.URL + `"}`
	w := doPing(t, body)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp PingResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success=true, got false")
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status_code=200, got %d", resp.StatusCode)
	}
	if resp.LatencyMs < 0 {
		t.Fatalf("latency should be >= 0, got %d", resp.LatencyMs)
	}
	if resp.Error != "" {
		t.Fatalf("expected empty error, got %q", resp.Error)
	}
}

func TestHandlePing_Non2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	body := `{"url":"` + srv.URL + `"}`
	w := doPing(t, body)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp PingResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected success=false for 500, got true")
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status_code=500, got %d", resp.StatusCode)
	}
}

func TestHandlePing_DefaultTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// omit timeout field — should use default 10s
	body := `{"url":"` + srv.URL + `"}`
	w := doPing(t, body)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp PingResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success=true, got false")
	}
}

func TestHandlePing_CustomTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	body := `{"url":"` + srv.URL + `","timeout":"5s"}`
	w := doPing(t, body)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp PingResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success=true, got false")
	}
}

func TestHandlePing_Timeout1sBoundary(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	body := `{"url":"` + srv.URL + `","timeout":"1s"}`
	w := doPing(t, body)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp PingResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success=true, got false")
	}
}

func TestHandlePing_Timeout30sBoundary(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	body := `{"url":"` + srv.URL + `","timeout":"30s"}`
	w := doPing(t, body)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp PingResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success=true, got false")
	}
}

func TestHandlePing_TimeoutError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	body := `{"url":"` + srv.URL + `","timeout":"1s"}`
	w := doPing(t, body)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp PingResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected success=false for timeout")
	}
	if !strings.HasPrefix(resp.Error, "timeout:") {
		t.Fatalf("expected error with 'timeout:' prefix, got %q", resp.Error)
	}
}

func TestHandlePing_TLSUnknownAuthority(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	body := `{"url":"` + srv.URL + `","timeout":"5s"}`
	w := doPing(t, body)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp PingResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected success=false for TLS error")
	}
	if resp.Error != "tls: unknown authority" {
		t.Fatalf("expected 'tls: unknown authority', got %q", resp.Error)
	}
}

func TestHandlePing_ConnectionRefused(t *testing.T) {
	// Start a listener on a random port, grab the address, then close it.
	// Pinging the closed port produces a connection-refused error.
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := l.Addr().String()
	l.Close()

	body := `{"url":"http://` + addr + `","timeout":"5s"}`
	w := doPing(t, body)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp PingResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected success=false for connection refused")
	}
	if !strings.HasPrefix(resp.Error, "connection:") {
		t.Fatalf("expected 'connection:' prefix, got %q", resp.Error)
	}
}
