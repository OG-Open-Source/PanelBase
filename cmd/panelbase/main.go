package main

import (
    "log"
    "github.com/OG-Open-Source/PanelBase/internal/server"
    "github.com/OG-Open-Source/PanelBase/internal/config"
)

func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    srv := server.New(cfg)
    if err := srv.Start(); err != nil {
        log.Fatalf("Server failed: %v", err)
    }
} 