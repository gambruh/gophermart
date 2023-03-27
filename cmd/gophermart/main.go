package main

import (
	"log"
	"net/http"

	"github.com/gambruh/gophermart/internal/accrualworker"
	"github.com/gambruh/gophermart/internal/auth"
	"github.com/gambruh/gophermart/internal/config"
	"github.com/gambruh/gophermart/internal/database"
	"github.com/gambruh/gophermart/internal/handlers"
)

func main() {
	config.InitFlags()
	config.SetConfig()
	authstorage := auth.GetAuthDB()
	defstorage := database.GetDB()

	service := handlers.NewService(defstorage, authstorage)
	agent := accrualworker.NewAgent(defstorage)

	server := &http.Server{
		Addr:    config.Cfg.Address,
		Handler: service.Service(),
	}

	go agent.CheckAccrual()

	log.Println(server.ListenAndServe())

}
