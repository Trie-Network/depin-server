package server

import (
	"net/http"
	"depin-server/constants"
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

	insertSQL := `
	INSERT INTO assets (
		id, name, main_category, secondary_category, description, depin_provider_did, metrics, category, owner
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);
	`
	_, err := s.DB.Exec(insertSQL, asset.ID, asset.Name, asset.MainCategory, asset.SecondaryCategory,
		asset.Description, asset.DepinProviderDID, asset.Metrics, asset.Category, asset.Owner)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, asset)
}

func (s *DepinServer) GetAssetsByCategory(c *gin.Context) {
	category := c.Query("category")
	if category != constants.ASSET_TYPE_MODEL && category != constants.ASSET_TYPE_DATASET {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category. Use 'model' or 'dataset'"})
		return
	}

	rows, err := s.DB.Query("SELECT * FROM assets WHERE category = ?", category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var assets []Asset
	for rows.Next() {
		var a Asset
		if err := rows.Scan(&a.ID, &a.Name, &a.MainCategory, &a.SecondaryCategory, &a.Description,
			&a.DepinProviderDID, &a.Metrics, &a.Category, &a.Owner); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		assets = append(assets, a)
	}

	c.JSON(http.StatusOK, assets)
}

func (s *DepinServer) GetAssetByID(c *gin.Context) {
	id := c.Param("id")
	row := s.DB.QueryRow("SELECT * FROM assets WHERE id = ?", id)

	var a Asset
	err := row.Scan(&a.ID, &a.Name, &a.MainCategory, &a.SecondaryCategory, &a.Description,
		&a.DepinProviderDID, &a.Metrics, &a.Category, &a.Owner)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Asset not found"})
		return
	}

	c.JSON(http.StatusOK, a)
}

func (s *DepinServer) GetAssetByDID(c *gin.Context) {
	did := c.Param("did")
	rows, err := s.DB.Query("SELECT * FROM assets WHERE depin_provider_did = ?", did)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var assets []Asset
	for rows.Next() {
		var a Asset
		if err := rows.Scan(&a.ID, &a.Name, &a.MainCategory, &a.SecondaryCategory, &a.Description,
			&a.DepinProviderDID, &a.Metrics, &a.Category, &a.Owner); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		assets = append(assets, a)
	}

	if len(assets) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No assets found for given DID"})
		return
	}

	c.JSON(http.StatusOK, assets)
}
