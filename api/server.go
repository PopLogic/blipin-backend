package api

import (
	"fmt"

	db "github.com/PopLogic/blipin-backend/db/sqlc"
	token "github.com/PopLogic/blipin-backend/token"
	"github.com/PopLogic/blipin-backend/util"
	"github.com/PopLogic/blipin-backend/worker"
	"github.com/gin-gonic/gin"
)

// Server serves HTTP requests for the API.
type Server struct {
	config          util.Config
	store           *db.Store
	router          *gin.Engine
	tokenMaker      token.Maker
	taskDistributor *worker.RedisTaskDistributor
}

func NewServer(config util.Config, store *db.Store, taskDistributor *worker.RedisTaskDistributor) (*Server, error) {
	tokenMaker, err := token.NewJWTMaker(config.JWTSecretKey) // Replace with your actual secret key
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	server := &Server{
		config:          config,
		store:           store,
		router:          gin.Default(),
		tokenMaker:      tokenMaker,
		taskDistributor: taskDistributor,
	}

	server.setupRouter()

	return server, nil
}

func (server *Server) setupRouter() {
	auth := server.router.Group("/auth")
	{
		auth.POST("/google", server.googleLogin)
		auth.POST("/apple", server.appleLogin)
		auth.POST("/email/login", server.emailLogin)
		// auth.GET("/email/verify", server.verifyEmail)
		auth.GET("/email/check", server.checkEmailAvailability)
	}

	token := server.router.Group("/token")
	{
		token.POST("/renew_access_token", server.renewAccessToken)
	}

	user := server.router.Group("/user")
	{
		user.PUT("", authMiddleware(server.tokenMaker), server.updateUserProfile)
		user.GET("", authMiddleware(server.tokenMaker), server.getUserProfile)
	}
}

func (server *Server) Start(address string) error {
	return server.router.Run(address)
}
