package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

var baseTmp *template.Template = template.Must(template.ParseFiles("templates/index.html"))

func main() {
	manager := NewManager()
	mux := mux.NewRouter()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { baseTmp.Execute(w, nil) })
	mux.HandleFunc("/room/{roomID}", manager.serverWS)
	server := &http.Server{
		Addr:         ":5000",
		Handler:      mux,
		ReadTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 5,
	}
	ctx, cancel := context.WithCancel(context.Background())
	manager.cancel = cancel
	go manager.run(ctx)
	defer func() {
		fmt.Println("main is closed")
	}()

	fmt.Println("Server is running on")
	log.Fatal(server.ListenAndServe())
}
