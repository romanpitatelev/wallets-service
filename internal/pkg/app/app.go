package app

import (
	"fmt"
	"go/projects/go-wallet/internal/app/endpoint"
	"go/projects/go-wallet/internal/app/service"
	"log"
	"net/http"
)

type App struct {
	e      *endpoint.Endpoint
	s      *service.Service
	server http.Server
}

func New() (*App, error) {
	a := &App{}
	a.s = service.New()
	a.e = endpoint.New(a.s)

	router := http.NewServeMux()
	router.HandleFunc("/time", a.e.Status)

	a.server = http.Server{
		Addr:    ":8081",
		Handler: router,
	}

	return a, nil

}

func (a *App) Run() error {
	fmt.Println("Server is listening on port 8081")
	err := a.server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}

	return nil
}
