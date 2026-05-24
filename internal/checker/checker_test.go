package checker

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/url"
	"testing"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		// ── timeout (os.IsTimeout) ──
		{
			name: "deadline exceeded (bare)",
			err:  context.DeadlineExceeded,
			want: "timeout: context deadline exceeded",
		},
		{
			name: "deadline exceeded wrapped in url.Error",
			err: &url.Error{
				Op:  "read",
				URL: "http://example.com",
				Err: context.DeadlineExceeded,
			},
			want: fmt.Sprintf("timeout: %s", (&url.Error{
				Op:  "read",
				URL: "http://example.com",
				Err: context.DeadlineExceeded,
			}).Error()),
		},
		{
			name: "DNS timeout (os.IsTimeout wins over dns: prefix)",
			err: &url.Error{
				Op:  "dial",
				URL: "http://example.com",
				Err: &net.DNSError{
					Err:       "timed out",
					Name:      "example.com",
					IsTimeout: true,
				},
			},
			want: fmt.Sprintf("timeout: %s", (&url.Error{
				Op:  "dial",
				URL: "http://example.com",
				Err: &net.DNSError{
					Err:       "timed out",
					Name:      "example.com",
					IsTimeout: true,
				},
			}).Error()),
		},

		// ── cancel ──
		{
			name: "context canceled",
			err:  context.Canceled,
			want: "canceled",
		},

		// ── invalid URL ──
		{
			name: "url parse error",
			err: &url.Error{
				Op:  "parse",
				URL: "http://[::1]:namedport",
				Err: errors.New("invalid port"),
			},
			want: "invalid_url: invalid port",
		},

		// ── DNS ──
		{
			name: "dns host not found",
			err: &url.Error{
				Op:  "dial",
				URL: "http://nonexistent.invalid",
				Err: &net.DNSError{
					Err:        "no such host",
					Name:       "nonexistent.invalid",
					IsNotFound: true,
				},
			},
			want: "dns: host not found",
		},
		{
			name: "dns temporary failure",
			err: &url.Error{
				Op:  "dial",
				URL: "http://example.com",
				Err: &net.DNSError{
					Err:         "server misbehaving",
					Name:        "example.com",
					IsTemporary: true,
				},
			},
			want: fmt.Sprintf("dns: %s", (&net.DNSError{
				Err:         "server misbehaving",
				Name:        "example.com",
				IsTemporary: true,
			}).Error()),
		},

		// ── TLS / certificates ──
		{
			name: "tls unknown authority",
			err: &url.Error{
				Op:  "dial",
				URL: "https://self-signed.example",
				Err: x509.UnknownAuthorityError{},
			},
			want: "tls: unknown authority",
		},
		{
			name: "tls hostname mismatch",
			err: &url.Error{
				Op:  "dial",
				URL: "https://wrong-host.example",
				Err: x509.HostnameError{Host: "wrong-host.example"},
			},
			want: "tls: hostname mismatch",
		},
		{
			name: "tls invalid certificate (expired)",
			err: &url.Error{
				Op:  "dial",
				URL: "https://expired.example",
				Err: x509.CertificateInvalidError{
					Reason: x509.Expired,
				},
			},
			want: "tls: invalid certificate",
		},
		{
			name: "tls certificate verification failed (generic)",
			err: &url.Error{
				Op:  "dial",
				URL: "https://example.com",
				Err: &tls.CertificateVerificationError{},
			},
			want: "tls: certificate verification failed",
		},

		// ── connection ──
		{
			name: "connection refused",
			err: &url.Error{
				Op:  "dial",
				URL: "http://localhost:9999",
				Err: &net.OpError{
					Op:  "dial",
					Net: "tcp",
					Err: errors.New("connection refused"),
				},
			},
			want: fmt.Sprintf("connection: %s", (&net.OpError{
				Op:  "dial",
				Net: "tcp",
				Err: errors.New("connection refused"),
			}).Error()),
		},

		// ── generic network fallback ──
		{
			name: "network fallback for unknown url.Error inner",
			err: &url.Error{
				Op:  "read",
				URL: "http://example.com",
				Err: errors.New("something unexpected"),
			},
			want: "network: something unexpected",
		},

		// ── bare unrecognized error ──
		{
			name: "bare error (not url.Error, not timeout)",
			err:  errors.New("some random error"),
			want: "some random error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classify(tt.err)
			if got != tt.want {
				t.Errorf("classify() = %q\nwant       = %q", got, tt.want)
			}
		})
	}
}
