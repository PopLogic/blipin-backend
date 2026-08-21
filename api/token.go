package api

import (
	"net/http"
	"time"

	db "github.com/PopLogic/blipin-backend/db/sqlc"
	"github.com/gin-gonic/gin"
)

type RenewAccessTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type RenewAccessTokenResponse struct {
	AccessToken           string `json:"access_token"`
	AccessTokenExpiresAt  time.Time `json:"access_token_expires_at"`
	RefreshToken          string `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`
}

func (server *Server) renewAccessToken(c *gin.Context) {
	var req RenewAccessTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payload, err := server.tokenMaker.VerifyToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	session, err := server.store.GetSessionByID(c, payload.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve session"})
		return
	}

	if session.IsBlocked {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "session is blocked"})
		return
	}

	if session.RefreshToken != req.RefreshToken {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token does not match"})
		return
	}

	if time.Now().After(session.ExpiresAt) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token has expired"})
		return
	}

	// rotate the refresh token
	newRefreshToken, refreshTokenPayload, err := server.tokenMaker.CreateToken(payload.UserID, server.config.RefreshTokenDuration)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create new refresh token"})
		return
	}

	// update the session with the new refresh token and expiration time
	updatedSession, err := server.store.UpdateSessionRefreshTokenAndExpiredAt(c, db.UpdateSessionRefreshTokenAndExpiredAtParams{
		ID:           payload.ID,
		RefreshToken: newRefreshToken,
		ExpiresAt:    refreshTokenPayload.ExpiresAt,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update session"})
		return
	}

	accessToken, accessTokenPayload, err := server.tokenMaker.CreateToken(payload.UserID, server.config.AccessTokenDuration)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create access token"})
		return
	}

	res := RenewAccessTokenResponse{
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessTokenPayload.ExpiresAt,
		RefreshToken:          updatedSession.RefreshToken,
		RefreshTokenExpiresAt: updatedSession.ExpiresAt,
	}

	c.JSON(http.StatusOK, res)

}
