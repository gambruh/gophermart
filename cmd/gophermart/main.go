package main

import (
	"net/http"
	"time"

	"github.com/gambruh/gophermart/internal/accrualworker"
	"github.com/gambruh/gophermart/internal/config"
	"github.com/gambruh/gophermart/internal/database"
	"github.com/gambruh/gophermart/internal/handlers"
	"github.com/gambruh/gophermart/internal/storage"
)

var service *handlers.WebService
var agent *accrualworker.Agent
var defstorage storage.Storage

func main() {
	config.InitFlags()
	config.SetConfig()

	if config.Cfg.Storage {
		defstorage = storage.NewStorage()
	} else {
		db := database.NewSQLdb(config.Cfg.Database)
		db.InitDatabase()
		defstorage = db
	}

	service = handlers.NewService(defstorage)
	agent = accrualworker.NewAgent(defstorage)

	server := &http.Server{
		Addr:    config.Cfg.Address,
		Handler: service.Service(),
	}

	//ask accrual for processed orders
	PingTime := time.NewTicker(900 * time.Millisecond)
	defer PingTime.Stop()
	go func() {
		for {
			<-PingTime.C
			agent.PingAccrual()
		}
	}()

	server.ListenAndServe()

}
