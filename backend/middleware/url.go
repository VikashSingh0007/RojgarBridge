package middleware

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"rojgarbridge-backend/config"
)

// DetectFrontendURL dynamically figures out the calling frontend domain (Vercel, custom domain, or localhost)
func DetectFrontendURL(r *http.Request, cfg *config.Config) string {
	if redirectParam := r.URL.Query().Get("redirect"); redirectParam != "" {
		if strings.HasPrefix(redirectParam, "http://") || strings.HasPrefix(redirectParam, "https://") {
			return strings.TrimRight(redirectParam, "/")
		}
	}

	if referer := r.Header.Get("Referer"); referer != "" {
		u, err := url.Parse(referer)
		if err == nil && u.Scheme != "" && u.Host != "" {
			return fmt.Sprintf("%s://%s", u.Scheme, u.Host)
		}
	}

	if origin := r.Header.Get("Origin"); origin != "" {
		return strings.TrimRight(origin, "/")
	}

	if cfg != nil && cfg.FrontendURL != "" && !strings.Contains(cfg.FrontendURL, "localhost") {
		return strings.TrimRight(cfg.FrontendURL, "/")
	}

	if cfg != nil && cfg.FrontendURL != "" {
		return strings.TrimRight(cfg.FrontendURL, "/")
	}
	return "http://localhost:3000"
}

// DetectBackendCallbackURL dynamically determines the callback URL (e.g. on production)
func DetectBackendCallbackURL(r *http.Request, cfg *config.Config) string {
	if cfg != nil && cfg.GoogleRedirectURL != "" && !strings.Contains(cfg.GoogleRedirectURL, "localhost") && !strings.Contains(cfg.GoogleRedirectURL, "127.0.0.1") {
		return cfg.GoogleRedirectURL
	}

	host := r.Host
	if host != "" && !strings.Contains(host, "localhost") && !strings.Contains(host, "127.0.0.1") {
		proto := "https"
		if fwd := r.Header.Get("X-Forwarded-Proto"); fwd != "" {
			proto = fwd
		}
		return fmt.Sprintf("%s://%s/api/auth/google/callback", proto, host)
	}

	if cfg != nil && cfg.GoogleRedirectURL != "" {
		return cfg.GoogleRedirectURL
	}
	return "http://localhost:8080/api/auth/google/callback"
}
