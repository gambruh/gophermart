package main

import (
	"net/http"

	"github.com/gambruh/gophermart/internal/config"
	"github.com/gambruh/gophermart/internal/database"
	"github.com/gambruh/gophermart/internal/handlers"
)

func main() {
	config.InitFlags()
	config.SetConfig()
	db := database.NewSQLdb(config.Cfg.Database)
	db.InitDatabase()

	service := handlers.NewService(db)

	server := &http.Server{
		Addr:    config.Cfg.Address,
		Handler: service.Service(),
	}

	server.ListenAndServe()
}
