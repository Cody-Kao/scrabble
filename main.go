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

var drawTmp *template.Template = template.Must(template.ParseFiles("templates/index.html"))
var homeTmp *template.Template = template.Must(template.ParseFiles("templates/home.html"))

func main() {
	manager := NewManager()
	mux := mux.NewRouter()
	// 用filerServer去存取static folder
	mux.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	mux.PathPrefix("/draw/static/").Handler(http.StripPrefix("/draw/static/", http.FileServer(http.Dir("./static"))))
	mux.HandleFunc("/", manager.home)
	//mux.HandleFunc("/draw/{roomID}", manager.getJoin).Methods("GET")
	mux.HandleFunc("/draw", manager.enter)
	mux.HandleFunc("/postJoin", manager.postCreateRoom).Methods("POST")
	mux.HandleFunc("/roomIDJoin", manager.postRoomIDJoin).Methods("POST")
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
