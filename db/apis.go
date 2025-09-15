package db

import (
	"encoding/json"
	"log"
	"net/http"
	"github.com/gorilla/mux"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

)

type Asset struct {
	ID                string `gorm:"column:id;primaryKey" json:"id"`
	Name              string `gorm:"column:name" json:"name"`
	MainCategory      string `gorm:"column:main_category" json:"main_category"`
	SecondaryCategory string `gorm:"column:secondary_category" json:"secondary_category"`
	Description       string `gorm:"column:description" json:"description"`
	DepinProviderDID  string `gorm:"column:depin_provider_did" json:"depin_provider_did"`
	Metrics           string `gorm:"column:metrics" json:"metrics"`
	Category          string `gorm:"column:category" json:"category"` 
	Owner			  string `gorm:"column:owner" json:"owner"`
}

var db *gorm.DB

func CreateDB() {
	var err error

	db, err = gorm.Open(sqlite.Open("assets.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to SQLite DB:", err)
	}
	db.AutoMigrate(&Asset{})
	r := mux.NewRouter()
	r.HandleFunc("/api/assets/create", CreateAsset).Methods("POST")
	r.HandleFunc("/api/assets", GetAssetsByCategory).Methods("GET")
	r.HandleFunc("/api/assets/{id}", GetAssetByID).Methods("GET")
	r.HandleFunc("/api/assets/did/{did}", GetAssetByDID).Methods("GET")

	log.Println("DApp Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func CreateAsset(w http.ResponseWriter, r *http.Request) {
	var asset Asset
	if err := json.NewDecoder(r.Body).Decode(&asset); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if asset.ID == "" || asset.Category == "" {
		http.Error(w, "ID and Category are required", http.StatusBadRequest)
		return
	}

	if err := db.Create(&asset).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func GetAssetsByCategory(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	if category != "model" && category != "dataset" {
		http.Error(w, "Invalid category. Use 'model' or 'dataset'", http.StatusBadRequest)
		return
	}

	var assets []Asset
	if err := db.Where("category = ?", category).Find(&assets).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(assets)
}

func GetAssetByID(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]

	var asset Asset
	if err := db.First(&asset, "id = ?", id).Error; err != nil {
		http.Error(w, "Asset not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(asset)
}

func GetAssetByDID(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	did := params["did"]

	var assets []Asset
	if err := db.Where("depin_provider_did = ?", did).Find(&assets).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(assets) == 0 {
		http.Error(w, "No assets found for given DID", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(assets)
}