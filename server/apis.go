package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *DepinServer) CreateAsset(c *gin.Context) {
	var asset Asset
	if err := c.ShouldBindJSON(&asset); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if asset.ID == "" || asset.Category == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID and Category are required"})
		return
	}

	if err := s.DB.Create(&asset).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, asset)
}

func (s *DepinServer) GetAssetsByCategory(c *gin.Context) {
	category := c.Query("category")
	if category != "model" && category != "dataset" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category. Use 'model' or 'dataset'"})
		return
	}

	var assets []Asset
	if err := s.DB.Where("category = ?", category).Find(&assets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, assets)
}

func (s *DepinServer) GetAssetByID(c *gin.Context) {
	id := c.Param("id")
	var asset Asset
	if err := s.DB.First(&asset, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Asset not found"})
		return
	}
	c.JSON(http.StatusOK, asset)
}

func (s *DepinServer) GetAssetByDID(c *gin.Context) {
	did := c.Param("did")
	var assets []Asset
	if err := s.DB.Where("depin_provider_did = ?", did).Find(&assets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(assets) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No assets found for given DID"})
		return
	}
	c.JSON(http.StatusOK, assets)
}
