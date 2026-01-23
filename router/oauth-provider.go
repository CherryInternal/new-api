package router

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/service/hydra"

	"github.com/gin-gonic/gin"
)

// SetOAuthProviderRouter sets up OAuth provider routes for Hydra login/consent/logout flows
// These routes are used when new-api acts as a Login/Consent Provider for Ory Hydra
func SetOAuthProviderRouter(router *gin.Engine) {
	if !common.HydraEnabled {
		return
	}

	// Initialize Hydra service
	hydraService := hydra.NewService(common.HydraAdminURL)
	ctrl := controller.NewOAuthProviderController(hydraService)

	// OAuth provider routes (called by Hydra)
	oauthRoute := router.Group("/oauth")
	oauthRoute.Use(middleware.GlobalAPIRateLimit())
	{
		// Login flow
		// GET /oauth/login - Hydra redirects here, shows login page or auto-accepts if session exists
		oauthRoute.GET("/login", ctrl.OAuthLogin)
		// POST /oauth/login - User submits login credentials
		oauthRoute.POST("/login", middleware.CriticalRateLimit(), ctrl.OAuthLoginSubmit)
		// POST /oauth/login/2fa - User submits 2FA code during OAuth login
		oauthRoute.POST("/login/2fa", middleware.CriticalRateLimit(), ctrl.OAuthLogin2FA)

		// Consent flow
		// GET /oauth/consent - Hydra redirects here, shows consent page or auto-accepts for trusted clients
		oauthRoute.GET("/consent", ctrl.OAuthConsent)
		// POST /oauth/consent - User grants consent with selected scopes
		oauthRoute.POST("/consent", ctrl.OAuthConsentSubmit)
		// POST /oauth/consent/reject - User rejects consent
		oauthRoute.POST("/consent/reject", ctrl.OAuthConsentReject)

		// Logout flow
		// GET /oauth/logout - Hydra redirects here, handles logout
		oauthRoute.GET("/logout", ctrl.OAuthLogout)
	}
}
