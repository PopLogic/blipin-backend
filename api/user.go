package api

import (
	"errors"
	"net/http"

	db "github.com/PopLogic/blipin-backend/db/sqlc"
	token "github.com/PopLogic/blipin-backend/token"
	"github.com/gin-gonic/gin"
)

type UpdateUserProfileRequest struct {
	DisplayName *string `json:"display_name"`
	Birthdate   *string `json:"birthdate"`
	Gender      *string `json:"gender"`
}

func (server *Server) getUserProfile(c *gin.Context) {
	// Get the user ID from the authorization payload set by the authMiddleware
	authPayload := c.MustGet(authorizationPayloadKey).(*token.Payload)

	// Call your store method to get the user profile
	userProfile, err := server.store.GetUser(c, authPayload.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": userProfile})
}

func (server *Server) checkEmailAvailability(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email query parameter is required"})
		return
	}

	// Call your store method to check if the email exists
	exists, err := server.store.CheckUserExistsByPasswordCredential(c, email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"is_available": exists == nil}) // If exists is nil, the email is available
}

func (server *Server) updateUserProfile(c *gin.Context) {
	var req UpdateUserProfileRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get the user ID from the authorization payload set by the authMiddleware
	authPayload := c.MustGet(authorizationPayloadKey).(*token.Payload)

	// Call your store method to update the user profile
	updatedUser, err := server.store.UpdateUserProfile(c, authPayload.UserID, req.DisplayName, req.Birthdate, req.Gender)
	if err != nil {
		if errors.Is(err, db.ErrInvalidBirthdate) || errors.Is(err, db.ErrInvalidGender) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully", "user": updatedUser})
}
