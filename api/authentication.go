package api

import (
	"fmt"
	"net/http"
	"time"

	db "github.com/PopLogic/blipin-backend/db/sqlc"
	"github.com/PopLogic/blipin-backend/worker"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"google.golang.org/api/idtoken"
)

type GoogleLoginRequest struct {
	IdToken string `json:"id_token" binding:"required"`
}

type GoogleLoginResponse struct {
	User                  db.User         `json:"user"`
	UserIdentity          db.UserIdentity `json:"user_identity"`
	AccessToken           string          `json:"access_token"`
	RefreshToken          string          `json:"refresh_token"`
	AccessTokenExpiresAt  time.Time       `json:"access_token_expires_at"`
	RefreshTokenExpiresAt time.Time       `json:"refresh_token_expires_at"`
	SessionID             uuid.UUID       `json:"session_id"`
	IsFirstTimeLogin      bool            `json:"is_first_time_login"`
}

func (server *Server) googleLogin(c *gin.Context) {
	// Check if the user is already logged in
	var req GoogleLoginRequest

	// Bind the JSON request body to the GoogleLoginRequest struct
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Validate the ID token using Google's ID token validator
	payload, err := idtoken.Validate(c, req.IdToken, "")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid ID token"})
		return
	}
	providerSub := payload.Claims["sub"].(string)
	name := payload.Claims["name"].(string)
	email := payload.Claims["email"].(string)
	picture := payload.Claims["picture"].(string)

	// Check if the user already exists by email (password credential)
	checkUserExistsByPasswordCredentialResult, err := server.store.CheckUserExistsByPasswordCredential(c, email)
	if err != nil {
		fmt.Println("Error checking user existence by password credential:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if checkUserExistsByPasswordCredentialResult != nil {
		fmt.Println("User already exists by password credential")
		c.JSON(http.StatusConflict, gin.H{
			"error": "An account with this email already exists and uses email/password login",
		})
		return
	}

	// Check if the user already exists by provider and provider_sub (user identity)
	checkUserExistsByUserIdentityResult, err := server.store.CheckUserExistsByUserIdentity(
		c,
		db.CheckUserExistsByUserIdentityParams{
			Provider:    string(db.AuthProviderGoogle),
			ProviderSub: providerSub,
		},
	)
	if err != nil {
		fmt.Println("Error checking user existence:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if checkUserExistsByUserIdentityResult != nil {
		fmt.Println("User already exists")
		// Create access and refresh tokens
		accessToken, accessTokenPayload, err := server.tokenMaker.CreateToken(checkUserExistsByUserIdentityResult.User.ID, server.config.AccessTokenDuration)
		if err != nil {
			fmt.Println("Error creating access token:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		refreshToken, refreshTokenPayload, err := server.tokenMaker.CreateToken(checkUserExistsByUserIdentityResult.User.ID, server.config.RefreshTokenDuration)
		if err != nil {
			fmt.Println("Error creating refresh token:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		session, err := server.store.CreateSession(c, db.CreateSessionParams{
			ID:           refreshTokenPayload.ID,
			UserID:       checkUserExistsByUserIdentityResult.User.ID,
			RefreshToken: refreshToken,
			UserAgent:    c.Request.UserAgent(),
			ClientIp:     c.ClientIP(),
			IsBlocked:    false,
			ExpiresAt:    refreshTokenPayload.ExpiresAt,
		})
		if err != nil {
			fmt.Println("Error creating session:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, GoogleLoginResponse{
			User:                  checkUserExistsByUserIdentityResult.User,
			UserIdentity:          checkUserExistsByUserIdentityResult.UserIdentity,
			AccessToken:           accessToken,
			RefreshToken:          refreshToken,
			AccessTokenExpiresAt:  accessTokenPayload.ExpiresAt,
			RefreshTokenExpiresAt: refreshTokenPayload.ExpiresAt,
			SessionID:             session.ID,
			IsFirstTimeLogin:      false,
		})
		return
	}

	// Create the user with user identity
	createUserWithUserIdentityResult, err := server.store.CreateUserWithUserIdentity(c,
		server.tokenMaker,
		server.config.AccessTokenDuration, server.config.RefreshTokenDuration,
		db.CreateUserWithUserIdentityParams{
			DisplayName:   name,
			AvatarUrl:     picture,
			ProviderSub:   providerSub,
			ProviderEmail: email,
			Provider:      db.AuthProviderGoogle,
		})
	if err != nil {
		fmt.Println("User creation failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	fmt.Println("User created")

	// Create access and refresh tokens
	accessToken, accessTokenPayload, err := server.tokenMaker.CreateToken(createUserWithUserIdentityResult.User.ID, server.config.AccessTokenDuration)
	if err != nil {
		fmt.Println("Error creating access token:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	refreshToken, refreshTokenPayload, err := server.tokenMaker.CreateToken(createUserWithUserIdentityResult.User.ID, server.config.RefreshTokenDuration)
	if err != nil {
		fmt.Println("Error creating refresh token:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	session, err := server.store.CreateSession(c, db.CreateSessionParams{
		ID:           uuid.Must(uuid.NewV4()),
		UserID:       createUserWithUserIdentityResult.User.ID,
		RefreshToken: refreshToken,
		UserAgent:    c.Request.UserAgent(),
		ClientIp:     c.ClientIP(),
		IsBlocked:    false,
		ExpiresAt:    refreshTokenPayload.ExpiresAt,
	})
	if err != nil {
		fmt.Println("Error creating session:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, GoogleLoginResponse{
		User:                  createUserWithUserIdentityResult.User,
		UserIdentity:          createUserWithUserIdentityResult.UserIdentity,
		AccessToken:           accessToken,
		RefreshToken:          refreshToken,
		IsFirstTimeLogin:      true,
		AccessTokenExpiresAt:  accessTokenPayload.ExpiresAt,
		RefreshTokenExpiresAt: refreshTokenPayload.ExpiresAt,
		SessionID:             session.ID,
	})
}

func (server *Server) appleLogin(c *gin.Context) {
	// Handle Apple login logic here
	c.JSON(http.StatusOK, gin.H{
		"message": "Apple login successful",
	})
}

type emailLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type emailLoginResponse struct {
	AccessToken  string  `json:"access_token"`
	RefreshToken string  `json:"refresh_token"`
	User         db.User `json:"user"`
}

func (server *Server) emailLogin(c *gin.Context) {
	var req emailLoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	taskPayload := &worker.PayloadSendVerifyEmail{
		Email: req.Email,
	}
	err := server.taskDistributor.DistributeTaskSendVerifyEmail(c, taskPayload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send verification email"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Verification email sent successfully"})

	// user, err := server.store.GetUserByEmail(c, req.Email)
	// if err != nil {
	// 	if err == sql.ErrNoRows {
	// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
	// 	} else {
	// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	// 	}
	// }

	// if user == nil {
	// 	c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
	// 	return
	// }

	// // Here you would typically verify the password using a hashing function
	// // For simplicity, let's assume the password is stored in plain text (not recommended)
	// if user.PasswordHash != req.Password {
	// 	c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
	// 	return
	// }

	// // Generate access and refresh tokens (you would implement this logic)
	// accessToken := "access_token"   // Replace with actual token generation logic
	// refreshToken := "refresh_token" // Replace with actual token generation logic

	// c.JSON(http.StatusOK, emailLoginResponse{
	// 	AccessToken:  accessToken,
	// 	RefreshToken: refreshToken,
	// 	User:         *user,
	// })
}
