package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
)

var upgrade = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var workers = make(map[string]*websocket.Conn)
var sendChannel = make(chan map[string]interface{})
var readChannel = make(chan ASGIResponse)

type ASGIResponse struct {
	headers map[string][]string
	body string
	status int
}

func main() {
	http.HandleFunc("/", requestHandler)
	http.HandleFunc("/workers", handleWs)

	fs := http.FileServer(http.Dir("static/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	go scheduleWorkers()
	go scheduleRecv()

	_ = http.ListenAndServe(":80", nil)
}


func handleWs(w http.ResponseWriter, r *http.Request) {
	conn, _ := upgrade.Upgrade(w, r, nil) // error ignored for sake of simplicity
	workers["abc"] = conn
}

func scheduleWorkers() {
	for {
		if workers["abc"] != nil {
			msg := <-sendChannel
			_ = workers["abc"].WriteJSON(msg)
		}
	}
}

func scheduleRecv() {
	for {
		if workers["abc"] != nil {
			_, msg, err := workers["abc"].ReadMessage()
			if err == nil {
				data := ASGIResponse{}
				err := json.Unmarshal(msg, &data)
				if err == nil {
					readChannel <- data
				}
			}
		}
	}
}

func requestHandler(w http.ResponseWriter, r *http.Request) {
	outgoing := map[string]interface{}{
		"method":  r.Method,
		"path":    r.URL,
		"remote":  r.RemoteAddr,
		"headers": r.Header,
		"form":    r.Form,
		"server":  "Sandman",
	}
	sendChannel <- outgoing
	val := <-readChannel
	_, _ = fmt.Fprint(w, val.body)
}