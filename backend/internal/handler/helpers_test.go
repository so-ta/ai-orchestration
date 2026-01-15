package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name           string
		xForwardedFor  string
		xRealIP        string
		remoteAddr     string
		expectedIP     string
	}{
		{
			name:       "X-Forwarded-For with single IP",
			xForwardedFor: "192.168.1.1",
			expectedIP: "192.168.1.1",
		},
		{
			name:       "X-Forwarded-For with multiple IPs",
			xForwardedFor: "192.168.1.1, 10.0.0.1, 172.16.0.1",
			expectedIP: "192.168.1.1",
		},
		{
			name:       "X-Forwarded-For with spaces",
			xForwardedFor: "  192.168.1.1  ",
			expectedIP: "192.168.1.1",
		},
		{
			name:       "X-Real-IP header",
			xRealIP:    "10.0.0.1",
			remoteAddr: "127.0.0.1:8080",
			expectedIP: "10.0.0.1",
		},
		{
			name:       "X-Real-IP with spaces",
			xRealIP:    "  10.0.0.1  ",
			remoteAddr: "127.0.0.1:8080",
			expectedIP: "10.0.0.1",
		},
		{
			name:       "RemoteAddr IPv4 with port",
			remoteAddr: "192.168.1.100:12345",
			expectedIP: "192.168.1.100",
		},
		{
			name:       "RemoteAddr IPv6 with port",
			remoteAddr: "[::1]:8080",
			expectedIP: "::1",
		},
		{
			name:       "RemoteAddr IPv6 full with port",
			remoteAddr: "[2001:db8::1]:8080",
			expectedIP: "2001:db8::1",
		},
		{
			name:       "RemoteAddr without port (fallback)",
			remoteAddr: "192.168.1.100",
			expectedIP: "192.168.1.100",
		},
		{
			name:       "X-Forwarded-For takes precedence over X-Real-IP",
			xForwardedFor: "192.168.1.1",
			xRealIP:    "10.0.0.1",
			remoteAddr: "127.0.0.1:8080",
			expectedIP: "192.168.1.1",
		},
		{
			name:       "X-Real-IP takes precedence over RemoteAddr",
			xRealIP:    "10.0.0.1",
			remoteAddr: "127.0.0.1:8080",
			expectedIP: "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}
			if tt.remoteAddr != "" {
				req.RemoteAddr = tt.remoteAddr
			}

			result := getClientIP(req)
			assert.Equal(t, tt.expectedIP, result)
		})
	}
}
