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

// OAuth paths that should be rewritten to use the request's host/scheme
var oauthRedirectPaths = []string{
	"/oauth/login",
	"/oauth/consent",
	"/oauth/logout",
}

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
		proxy := createHydraProxy(target, c.Request)
		proxy.ServeHTTP(c.Writer, c.Request)
	})
	router.Any("/.well-known/*any", func(c *gin.Context) {
		proxy := createHydraProxy(target, c.Request)
		proxy.ServeHTTP(c.Writer, c.Request)
	})
}

// createHydraProxy creates a reverse proxy with automatic URL rewriting for OAuth redirects.
func createHydraProxy(target *url.URL, originalReq *http.Request) *httputil.ReverseProxy {
	requestHost := originalReq.Host
	requestScheme := getRequestScheme(originalReq)

	proxy := httputil.NewSingleHostReverseProxy(target)
	defaultDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		defaultDirector(req)
		// Pass original request info to Hydra
		if requestHost != "" {
			req.Header.Set("X-Forwarded-Host", requestHost)
		}
		if requestScheme != "" {
			req.Header.Set("X-Forwarded-Proto", requestScheme)
		}
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, proxyErr error) {
		common.SysLog(fmt.Sprintf("hydra proxy error: %s %s -> %v", r.Method, r.URL.String(), proxyErr))
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}

	// Rewrite OAuth redirect URLs to use the current request's host/scheme
	proxy.ModifyResponse = func(resp *http.Response) error {
		return rewriteOAuthRedirect(resp, requestHost, requestScheme)
	}

	return proxy
}

// getRequestScheme determines the scheme from the original request.
func getRequestScheme(req *http.Request) string {
	// Check X-Forwarded-Proto first (most common)
	if proto := req.Header.Get("X-Forwarded-Proto"); proto != "" {
		return strings.ToLower(strings.TrimSpace(proto))
	}
	// Check TLS
	if req.TLS != nil {
		return "https"
	}
	return "http"
}

// rewriteOAuthRedirect rewrites OAuth redirect URLs to use the request's host/scheme.
func rewriteOAuthRedirect(resp *http.Response, requestHost, requestScheme string) error {
	if resp.StatusCode < 300 || resp.StatusCode >= 400 {
		return nil
	}

	location := resp.Header.Get("Location")
	if location == "" {
		return nil
	}

	locURL, err := url.Parse(location)
	if err != nil || locURL.Host == "" {
		return nil
	}

	// Only rewrite OAuth paths
	if !isOAuthPath(locURL.Path) {
		return nil
	}

	// Rewrite to request's host/scheme
	oldLocation := location
	locURL.Host = requestHost
	locURL.Scheme = requestScheme

	resp.Header.Set("Location", locURL.String())
	common.SysLog(fmt.Sprintf("hydra rewrite: %s -> %s", oldLocation, locURL.String()))

	return nil
}

func isOAuthPath(path string) bool {
	for _, p := range oauthRedirectPaths {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}
