package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// getUserFromContext extracts and validates user ID from context.
// Returns 0 if unauthorized (also sends error response).
func getUserFromContext(c *gin.Context) int {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "unauthorized",
		})
	}
	return userId
}

// OAuthGetUserInfo returns user information for OAuth clients
// Scope required: openid, profile
func OAuthGetUserInfo(c *gin.Context) {
	userId := getUserFromContext(c)
	if userId == 0 {
		return
	}

	user, err := model.GetUserById(userId, false)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "user not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":           user.Id,
			"username":     user.Username,
			"display_name": user.DisplayName,
			"email":        user.Email,
			"group":        user.Group,
		},
	})
}

// OAuthGetBalance returns user balance information for OAuth clients
// Scope required: balance:read
func OAuthGetBalance(c *gin.Context) {
	userId := getUserFromContext(c)
	if userId == 0 {
		return
	}

	user, err := model.GetUserById(userId, false)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "user not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"quota":      user.Quota,
			"used_quota": user.UsedQuota,
		},
	})
}

// OAuthGetUsage returns user usage statistics for OAuth clients
// Scope required: usage:read
func OAuthGetUsage(c *gin.Context) {
	userId := getUserFromContext(c)
	if userId == 0 {
		return
	}

	user, err := model.GetUserById(userId, false)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "user not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"request_count": user.RequestCount,
			"used_quota":    user.UsedQuota,
			"quota":         user.Quota,
		},
	})
}

// OAuthListTokens returns user's API tokens for OAuth clients
// Scope required: tokens:read
func OAuthListTokens(c *gin.Context) {
	userId := getUserFromContext(c)
	if userId == 0 {
		return
	}

	tokens, err := model.GetAllUserTokens(userId, 0, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to get tokens",
		})
		return
	}

	// Hide sensitive key data
	safeTokens := make([]gin.H, 0, len(tokens))
	for _, t := range tokens {
		safeTokens = append(safeTokens, gin.H{
			"id":              t.Id,
			"name":            t.Name,
			"status":          t.Status,
			"created_time":    t.CreatedTime,
			"expired_time":    t.ExpiredTime,
			"remain_quota":    t.RemainQuota,
			"unlimited_quota": t.UnlimitedQuota,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    safeTokens,
	})
}

// OAuthCreateToken creates a new API token for OAuth clients
// Scope required: tokens:write
func OAuthCreateToken(c *gin.Context) {
	userId := getUserFromContext(c)
	if userId == 0 {
		return
	}

	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid request: " + err.Error(),
		})
		return
	}

	// Validate name length
	if len(req.Name) > 30 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "token name too long (max 30 characters)",
		})
		return
	}

	// Generate key
	key, err := common.GenerateKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to generate token key",
		})
		return
	}

	// Create token with default settings
	token := &model.Token{
		UserId:         userId,
		Name:           req.Name,
		Key:            key,
		CreatedTime:    common.GetTimestamp(),
		AccessedTime:   common.GetTimestamp(),
		ExpiredTime:    -1, // Never expires
		UnlimitedQuota: false,
	}

	err = token.Insert()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to create token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":   token.Id,
			"name": token.Name,
			"key":  token.Key, // Return key only on creation
		},
	})
}

// OAuthDeleteToken deletes an API token for OAuth clients
// Scope required: tokens:write
func OAuthDeleteToken(c *gin.Context) {
	userId := getUserFromContext(c)
	if userId == 0 {
		return
	}

	tokenIdStr := c.Param("id")
	if tokenIdStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "missing token id",
		})
		return
	}

	tokenId, err := strconv.Atoi(tokenIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid token id",
		})
		return
	}

	// Verify token belongs to user
	token, err := model.GetTokenById(tokenId)
	if err != nil || token.UserId != userId {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "token not found",
		})
		return
	}

	err = model.DeleteTokenById(tokenId, userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to delete token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "token deleted",
	})
}
