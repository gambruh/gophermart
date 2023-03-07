package main

import (
	"net/http"

	"github.com/gambruh/gophermart/internal/config"
	"github.com/gambruh/gophermart/internal/database"
	"github.com/gambruh/gophermart/internal/handlers"
	"github.com/gambruh/gophermart/internal/storage"
)

func main() {
	config.InitFlags()
	config.SetConfig()
	var service *handlers.WebService

	if config.Cfg.Storage {
		memstorage := storage.NewStorage()
		service = handlers.NewService(&memstorage)
	} else {
		db := database.NewSQLdb(config.Cfg.Database)
		db.InitDatabase()
		service = handlers.NewService(db)
	}

	server := &http.Server{
		Addr:    config.Cfg.Address,
		Handler: service.Service(),
	}

	server.ListenAndServe()
}
