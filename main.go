package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"rugby-live-api/config"
	"rugby-live-api/db"
	"rugby-live-api/handlers"
	"rugby-live-api/services"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func main() {
	// Load environment variables
	if err := config.LoadConfig(); err != nil {
		log.Fatal("Error loading config:", err)
	}

	// Initialize database
	database, err := config.InitDB()
	if err != nil {
		log.Fatal("Error connecting to database:", err)
	}
	defer database.Close()

	// Create store and API client
	store := db.NewStore(sqlx.NewDb(database, "postgres"))
	apiClient := services.NewAPIClient()

	if len(os.Args) > 1 && os.Args[1] == "migrate-storage" {
		fmt.Println("Migrating storage paths...")
		if err := apiClient.MigrateStoragePaths(store); err != nil {
			log.Fatalf("Failed to migrate storage: %v", err)
		}
		fmt.Println("Storage migration complete")
		return
	}

	// Initialize router
	router := gin.Default()

	// Initialize handlers
	h := handlers.NewHandler(store)

	// Define routes
	router.GET("/matches", h.GetMatches)
	router.GET("/countries", h.GetCountries)
	router.POST("/countries/refresh", h.RefreshCountries)
	router.POST("/leagues/refresh", h.RefreshLeagues)
	router.POST("/teams/refresh", h.RefreshTeams)
	router.POST("/teams/update-images", h.UpdateTeamImages)
	router.GET("/espn/leagues", h.GetESPNLeagues)
	router.GET("/wikidata/teams", h.GetWikidataTeams)
	router.GET("/wikidata/teams/search", h.SearchWikidataTeams)
	router.POST("/rugbydb/teams", h.GetRugbyDBTeams)
	router.GET("/rugbydb/leagues/:year", func(c *gin.Context) {
		yearStr := c.Param("year")
		dryRun := c.DefaultQuery("dry_run", "false")
		isDryRun := dryRun == "true"
		leagues, err := apiClient.GetLeaguesByYear(store, yearStr, isDryRun)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, leagues)
	})

	// Start server
	router.Run(":8080")
}
