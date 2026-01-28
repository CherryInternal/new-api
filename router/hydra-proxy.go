package router

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"github.com/gin-gonic/gin"
)

// SetHydraPublicProxyRouter proxies Hydra public endpoints through new-api.
func SetHydraPublicProxyRouter(router *gin.Engine) {
	if !common.HydraEnabled {
		return
	}

	publicURL := strings.TrimSpace(common.HydraPublicURL)
	if publicURL == "" {
		return
	}

	target, err := url.Parse(publicURL)
	if err != nil || target.Scheme == "" || target.Host == "" {
		common.SysLog("invalid HYDRA_PUBLIC_URL: " + publicURL)
		return
	}

	router.Any("/oauth2/*any", func(c *gin.Context) {
		proxy := createHydraProxy(target, c.Request.Host, getRequestScheme(c.Request))
		proxy.ServeHTTP(c.Writer, c.Request)
	})
	router.Any("/.well-known/*any", func(c *gin.Context) {
		proxy := createHydraProxy(target, c.Request.Host, getRequestScheme(c.Request))
		proxy.ServeHTTP(c.Writer, c.Request)
	})
}

// createHydraProxy creates a reverse proxy with URL rewriting support for multi-domain setup.
func createHydraProxy(target *url.URL, requestHost, requestScheme string) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)
	defaultDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		defaultDirector(req)
		if requestHost != "" {
			req.Header.Set("X-Forwarded-Host", requestHost)
		}
		if requestScheme != "" {
			req.Header.Set("X-Forwarded-Proto", requestScheme)
		}
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, proxyErr error) {
		common.SysLog(fmt.Sprintf("hydra public proxy error: method=%s url=%s host=%s target=%s err=%v", r.Method, r.URL.String(), r.Host, target.String(), proxyErr))
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}

	// If HYDRA_BASE_HOST is configured, rewrite redirect URLs to use the current request's host
	baseHost := strings.TrimSpace(common.HydraBaseHost)
	if baseHost != "" && requestHost != "" && requestHost != baseHost {
		proxy.ModifyResponse = func(resp *http.Response) error {
			return rewriteRedirectLocation(resp, baseHost, requestHost)
		}
	}

	return proxy
}

func getRequestScheme(req *http.Request) string {
	if req == nil {
		return ""
	}
	if forwardedProto := parseForwardedProto(req.Header.Get("Forwarded")); forwardedProto != "" {
		return forwardedProto
	}
	if cfVisitorScheme := parseCFVisitorScheme(req.Header.Get("CF-Visitor")); cfVisitorScheme != "" {
		return cfVisitorScheme
	}
	if forwardedProto := strings.TrimSpace(req.Header.Get("X-Forwarded-Proto")); forwardedProto != "" {
		return forwardedProto
	}
	if req.TLS != nil {
		return "https"
	}
	return "http"
}

func parseForwardedProto(forwarded string) string {
	forwarded = strings.TrimSpace(forwarded)
	if forwarded == "" {
		return ""
	}
	for _, entry := range strings.Split(forwarded, ",") {
		for _, part := range strings.Split(entry, ";") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			kv := strings.SplitN(part, "=", 2)
			if len(kv) != 2 {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(kv[0]), "proto") {
				proto := strings.Trim(strings.TrimSpace(kv[1]), "\"")
				if proto != "" {
					return strings.ToLower(proto)
				}
			}
		}
	}
	return ""
}

func parseCFVisitorScheme(cfVisitor string) string {
	cfVisitor = strings.TrimSpace(cfVisitor)
	if cfVisitor == "" {
		return ""
	}
	if strings.Contains(cfVisitor, "\"scheme\":\"https\"") {
		return "https"
	}
	if strings.Contains(cfVisitor, "\"scheme\":\"http\"") {
		return "http"
	}
	return ""
}

// rewriteRedirectLocation rewrites the Location header in redirect responses
// to replace the base host with the current request's host.
func rewriteRedirectLocation(resp *http.Response, baseHost, requestHost string) error {
	// Only process redirect responses
	if resp.StatusCode < 300 || resp.StatusCode >= 400 {
		return nil
	}

	location := resp.Header.Get("Location")
	if location == "" {
		return nil
	}

	// Parse the location URL
	locURL, err := url.Parse(location)
	if err != nil {
		return nil // Don't fail on parse error, just skip rewriting
	}

	// Check if the location host matches the base host (with or without port)
	locHost := locURL.Host
	if locHost == "" {
		return nil // Relative URL, no rewriting needed
	}

	// Strip port from hosts for comparison
	locHostWithoutPort := stripPort(locHost)
	baseHostWithoutPort := stripPort(baseHost)

	if locHostWithoutPort == baseHostWithoutPort {
		// Rewrite the host to the request host
		locURL.Host = requestHost

		// Determine scheme based on request (assume HTTPS for production)
		if !strings.HasPrefix(requestHost, "localhost") && !strings.HasPrefix(requestHost, "127.0.0.1") {
			locURL.Scheme = "https"
		}

		resp.Header.Set("Location", locURL.String())
	}

	return nil
}

// stripPort removes the port from a host string.
func stripPort(host string) string {
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		// Make sure it's not an IPv6 address
		if strings.Contains(host, "]") {
			// IPv6: [::1]:8080
			if bracketIdx := strings.LastIndex(host, "]"); bracketIdx < idx {
				return host[:idx]
			}
		} else {
			return host[:idx]
		}
	}
	return host
}
