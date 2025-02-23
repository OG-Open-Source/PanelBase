package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"github.com/OG-Open-Source/PanelBase/internal/handlers"
	"github.com/OG-Open-Source/PanelBase/pkg/utils"
	"github.com/joho/godotenv"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	if err := utils.InitLogger(); err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}

	dirs := []string{
		"internal/commands",
		"internal/config",
		"logs",
		"themes",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatal("Failed to create directory:", dir, err)
		}
	}
}

func main() {
	utils.Log(utils.INFO, "PanelBase starting...")

	handler := handlers.NewExternalHandler()
	if err := handler.Init(); err != nil {
		utils.Log(utils.ERROR, "Failed to initialize handler: %v", err)
		os.Exit(1)
	}

	port := os.Getenv("PORT")
	if port == "" {
		utils.Log(utils.ERROR, "PORT environment variable not set")
		os.Exit(1)
	}

	entry := os.Getenv("ENTRY")
	if entry == "" {
		utils.Log(utils.ERROR, "ENTRY environment variable not set")
		os.Exit(1)
	}

	http.Handle("/"+entry+"/", handler)
	http.Handle("/", http.FileServer(http.Dir("web")))

	addr := fmt.Sprintf(":%s", port)
	utils.Log(utils.INFO, "Server starting on port %s with entry point /%s", port, entry)
	if err := http.ListenAndServe(addr, nil); err != nil {
		utils.Log(utils.ERROR, "Server failed to start: %v", err)
		os.Exit(1)
	}
} 