package checker

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Result struct {
	Success    bool
	StatusCode int
	Latency    time.Duration
	Error      string
}

// Ping sends a GET to url and returns a structured Result.
// It classifies the outcome: success (2xx), timeout, or other error.
func Ping(url string, timeout time.Duration) Result {
	client := &http.Client{
		Timeout: timeout,
	}

	start := time.Now()
	resp, err := client.Get(url)
	latency := time.Since(start)

	if err != nil {
		return Result{
			Success: false,
			Latency: latency,
			Error:   classify(err),
		}
	}
	defer resp.Body.Close()

	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	return Result{
		Success:    success,
		StatusCode: resp.StatusCode,
		Latency:    latency,
	}
}

func classify(err error) string {
	// Timeout — catch before unwrapping so both *url.Error and bare
	// context.DeadlineExceeded are classified uniformly.
	if os.IsTimeout(err) {
		return "timeout: " + err.Error()
	}
	if errors.Is(err, context.Canceled) {
		return "canceled"
	}

	urlErr, ok := errors.AsType[*url.Error](err)
	if !ok {
		return err.Error()
	}

	inner := urlErr.Err

	// Malformed URL.
	if urlErr.Op == "parse" {
		return "invalid_url: " + inner.Error()
	}

	// DNS failures (non-timeout — timeouts are caught above).
	if dnsErr, ok := errors.AsType[*net.DNSError](inner); ok {
		if dnsErr.IsNotFound {
			return "dns: host not found"
		}
		return "dns: " + dnsErr.Error()
	}

	// TLS / certificate errors — check specific x509 errors before the
	// generic *tls.CertificateVerificationError wrapper.
	if _, ok := errors.AsType[x509.UnknownAuthorityError](inner); ok {
		return "tls: unknown authority"
	}
	if _, ok := errors.AsType[x509.HostnameError](inner); ok {
		return "tls: hostname mismatch"
	}
	if _, ok := errors.AsType[x509.CertificateInvalidError](inner); ok {
		return "tls: invalid certificate"
	}
	if _, ok := errors.AsType[*tls.CertificateVerificationError](inner); ok {
		return "tls: certificate verification failed"
	}

	// Connection-level errors (non-timeout).
	if opErr, ok := errors.AsType[*net.OpError](inner); ok {
		return "connection: " + opErr.Error()
	}

	return "network: " + inner.Error()
}
