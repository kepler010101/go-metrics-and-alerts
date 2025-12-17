package middleware

import (
	"net"
	"net/http"
	"strings"
)

func WithTrustedSubnet(subnet *net.IPNet) func(http.Handler) http.Handler {
	if subnet == nil {
		return func(next http.Handler) http.Handler { return next }
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				next.ServeHTTP(w, r)
				return
			}

			if !strings.HasPrefix(r.URL.Path, "/update") && !strings.HasPrefix(r.URL.Path, "/updates") {
				next.ServeHTTP(w, r)
				return
			}

			ip := net.ParseIP(strings.TrimSpace(r.Header.Get("X-Real-IP")))
			if ip == nil || !subnet.Contains(ip) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
