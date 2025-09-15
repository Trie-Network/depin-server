package server

import (
	"depin-server/db"
	"depin-server/utils"
	"os"

	"github.com/gin-gonic/gin"
)

type DepinServer struct {
	Port             string
	Storage          *db.InferenceStorage
	RubixNodeAddress string

	router *gin.Engine
}

func NewDepinServer(port string, storage *db.InferenceStorage, rubixNodeAddress string) *DepinServer {
	depinServer := &DepinServer{
		Port:             port,
		Storage:          storage,
		RubixNodeAddress: rubixNodeAddress,
	}

	// Register DePIN server API routes
	depinServer.router = gin.Default()
	depinServer.registerRoutes()

	return depinServer
}

func (s *DepinServer) registerRoutes() {
	if s.router == nil {
		s.router = gin.Default()
	}

	apiV1 := s.router.Group("/depin-server/v1")
	{
		apiV1.GET("/healthz", s.HandleHealthCheck)
		if os.Getenv("ENABLE_ASSET_UPLOAD") == "true" {
			apiV1.POST("/upload", s.HandleFileUpload)
			apiV1.POST("/inference", s.HandleInference)
			apiV1.GET("/assets", s.HandleGetAssets)
			apiV1.GET("/assets/download/:assetId", s.HandleDownloadAsset)
		} else {
			utils.LogInfo("Depin Server is not accepting new assets, set ENABLE_ASSET_UPLOAD to true to allow uploads")
		}
	}
}

func (s *DepinServer) Start() error {
	if err := s.router.Run(":" + s.Port); err != nil {
		return err
	}
	return nil
}
