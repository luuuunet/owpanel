package main

import (
	"log"
	"strconv"

	"github.com/open-panel/open-panel/internal/api"
	"github.com/open-panel/open-panel/internal/config"
	"github.com/open-panel/open-panel/internal/database"
	"github.com/open-panel/open-panel/internal/services/settings"
)

func main() {
	cfg := config.Load()
	db, err := database.Init(cfg.DataDir)
	if err != nil {
		log.Fatalf("database init: %v", err)
	}

	settingsSvc := settings.NewServiceWithDataDir(db, cfg.DataDir)
	if all, err := settingsSvc.GetAll(); err == nil {
		if p := all["panel_port"]; p != "" {
			if n, err := strconv.Atoi(p); err == nil && n > 0 {
				cfg.Port = n
			}
		}
	}

	server := api.NewServer(cfg, db)
	if err := server.Run(); err != nil {
		log.Fatalf("server: %v", err)
	}
}
