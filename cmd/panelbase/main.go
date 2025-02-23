package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"github.com/OG-Open-Source/PanelBase/internal/handlers"
	"github.com/OG-Open-Source/PanelBase/pkg/utils"
	"github.com/joho/godotenv"
)

func init() {
	if err := os.MkdirAll("logs", 0755); err != nil {
		log.Fatalf("Failed to create logs directory: %v", err)
	}

	if err := utils.InitLogger(); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	if err := godotenv.Load(); err != nil {
		utils.Log(utils.EROR, "Failed to load .env file: %v", err)
		os.Exit(1)
	}

	requiredEnvVars := []string{"PORT", "ENTRY"}
	for _, env := range requiredEnvVars {
		if os.Getenv(env) == "" {
			utils.Log(utils.EROR, "Required environment variable %s is not set", env)
			os.Exit(1)
		}
	}

	dirs := []string{
		"internal/commands",
		"internal/config",
		"web",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			utils.Log(utils.EROR, "Failed to create directory %s: %v", dir, err)
			os.Exit(1)
		}
	}

	requiredFiles := []string{
		"internal/config/routes.json",
		"internal/commands/time.go",
		"internal/commands/run.sh",
	}

	for _, file := range requiredFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			utils.Log(utils.EROR, "Required file %s does not exist", file)
			os.Exit(1)
		}
	}
}

func main() {
	utils.Log(utils.INFO, "PanelBase starting...")

	handler := handlers.NewExternalHandler()
	if err := handler.Init(); err != nil {
		utils.Log(utils.EROR, "Failed to initialize handler: %v", err)
		os.Exit(1)
	}

	ip := os.Getenv("IP")
	if ip == "" {
		utils.Log(utils.WARN, "IP environment variable not set, using localhost")
		ip = "localhost"
	}

	port := os.Getenv("PORT")
	if port == "" {
		utils.Log(utils.EROR, "PORT environment variable not set")
		os.Exit(1)
	}

	entry := os.Getenv("ENTRY")
	if entry == "" {
		utils.Log(utils.EROR, "ENTRY environment variable not set")
		os.Exit(1)
	}

	fmt.Print("\033[2J")
	fmt.Print("\033[?25l")
	defer fmt.Print("\033[?25h")

	updateDisplay := func() {
		fmt.Print("\033[2J")
		fmt.Print("\033[H")
		fmt.Print("\033[J")
		fmt.Print("\033[?7l")
		defer fmt.Print("\033[?7h")

		fmt.Println("============================================")
		fmt.Println("PanelBase Agent Connection Details")
		fmt.Println("--------------------------------------------")
		fmt.Printf("- IP: \t\t%s\n", ip)
		fmt.Printf("- Port: \t%s\n", port)
		fmt.Printf("- Entry: \t%s\n", entry)
		fmt.Println("============================================")
		fmt.Println("Connection History")
		fmt.Println("============================================")

		logFile := filepath.Join("logs", fmt.Sprintf("%s.log", time.Now().Format("2006-01-02")))
		if logs, err := ioutil.ReadFile(logFile); err == nil {
			lines := strings.Split(string(logs), "\n")
			maxLines := 20
			startIndex := len(lines) - 1
			if startIndex > maxLines {
				startIndex = maxLines
			}
			for i := startIndex; i >= 0; i-- {
				line := strings.TrimSpace(lines[i])
				if line != "" {
					fmt.Printf(" %s\n", line)
				}
			}
		}
	}

	updateDisplay()

	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				updateDisplay()
			}
		}
	}()

	http.Handle("/"+entry+"/", handler)
	http.Handle("/", http.FileServer(http.Dir("web")))

	addr := fmt.Sprintf(":%s", port)
	utils.Log(utils.INFO, "Server starting on port %s with entry point /%s", port, entry)
	if err := http.ListenAndServe(addr, nil); err != nil {
		utils.Log(utils.EROR, "Server failed to start: %v", err)
		os.Exit(1)
	}
}