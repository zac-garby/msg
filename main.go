package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Zac-Garby/msg/server"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var s *server.Server

func main() {
	s = server.New()
	go s.HandleMessages()
	go s.HandleInput(os.Stdin)

	r := mux.NewRouter()
	r.HandleFunc("/", indexHandler)
	r.HandleFunc(`/room/{room:[\p{L}\p{N}-_./<>&]{1,64}}`, roomHandler)
	r.HandleFunc("/ws", websocketHandler)
	r.HandleFunc("/validate", validateHandler)

	r.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/",
			http.FileServer(http.Dir("./static/")),
		),
	)

	r.Handle("/favicon.ico", http.FileServer(http.Dir("./static/")))

	fmt.Println("listening on localhost:3000")
	http.ListenAndServe(":3000", r)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "max-age=0")

	if name, err := r.Cookie("name"); err == nil && name.Value != "" {
		http.ServeFile(w, r, "static/messager.html")
	} else {
		http.ServeFile(w, r, "static/index.html")
	}
}

func roomHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "max-age=0")
	room := mux.Vars(r)["room"]

	http.SetCookie(w, &http.Cookie{
		Name:   "room",
		Value:  room,
		Path:   "/",
		MaxAge: 30, // expires in 30s, since it's only used briefly
	})

	http.Redirect(w, r, "/", http.StatusFound)
}

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	if err := s.NewClient(conn); err != nil {
		log.Println("server err:", err)
		return
	}
}

func validateHandler(w http.ResponseWriter, r *http.Request) {
	vals := r.URL.Query()

	nameSlice, ok := vals["name"]
	if ok && len(nameSlice) > 0 {
		name := nameSlice[0]
		reason, valid := server.ValidateName(name, s)
		if !valid {
			fmt.Fprintf(w, reason)
			return
		}
	}

	roomSlice, ok := vals["room"]
	if ok && len(roomSlice) > 0 {
		room := roomSlice[0]
		reason, valid := server.ValidateRoom(room)
		if !valid {
			fmt.Fprintf(w, reason)
			return
		}
	}

	fmt.Fprintf(w, "ok")
}
