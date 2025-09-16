package server

import (
	"depin-server/db"
	"depin-server/utils"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type DepinServer struct {
	Port             string
	Storage          *db.InferenceStorage
	RubixNodeAddress string
	DB               *gorm.DB
	router           *gin.Engine
}

type Asset struct {
	ID                string `gorm:"column:id;primaryKey" json:"id"`
	Name              string `gorm:"column:name" json:"name"`
	MainCategory      string `gorm:"column:main_category" json:"main_category"`
	SecondaryCategory string `gorm:"column:secondary_category" json:"secondary_category"`
	Description       string `gorm:"column:description" json:"description"`
	DepinProviderDID  string `gorm:"column:depin_provider_did" json:"depin_provider_did"`
	Metrics           string `gorm:"column:metrics" json:"metrics"`
	Category          string `gorm:"column:category" json:"category"`
	Owner             string `gorm:"column:owner" json:"owner"`
}

// InitDB initializes SQLite and returns *gorm.DB
func InitDB(dbPath string) *gorm.DB {
	dbConn, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to SQLite DB:", err)
	}
	if err := dbConn.AutoMigrate(&Asset{}); err != nil {
		log.Fatal("Failed to migrate Asset schema:", err)
	}
	return dbConn
}

func NewDepinServer(port string, storage *db.InferenceStorage, rubixNodeAddress string, dbConn *gorm.DB) *DepinServer {
	s := &DepinServer{
		Port:             port,
		Storage:          storage,
		RubixNodeAddress: rubixNodeAddress,
		DB:               dbConn,
		router:           gin.Default(),
	}
	s.registerRoutes()
	return s
}

func (s *DepinServer) registerRoutes() {
	apiV1 := s.router.Group("/depin-server/v1")
	{
		apiV1.GET("/healthz", s.HandleHealthCheck)

		if os.Getenv("ENABLE_ASSET_UPLOAD") == "true" {
			apiV1.POST("/upload", s.HandleFileUpload)
			apiV1.POST("/inference", s.HandleInference)
			apiV1.GET("/assets", s.HandleGetAssets)
			apiV1.GET("/assets/download/:assetId", s.HandleDownloadAsset)

			// Asset APIs
			apiV1.POST("/assets/create", s.CreateAsset)
			apiV1.GET("/assets/category", s.GetAssetsByCategory)
			apiV1.GET("/assets/:id", s.GetAssetByID)
			apiV1.GET("/assets/did/:did", s.GetAssetByDID)
		} else {
			utils.LogInfo("Depin Server is not accepting new assets, set ENABLE_ASSET_UPLOAD to true to allow uploads")
		}
	}
}

func (s *DepinServer) Start() error {
	return s.router.Run(":" + s.Port)
}
