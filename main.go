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

func main() {
	http.HandleFunc("/", requestHandler)
	http.HandleFunc("/workers", handleWs)

	fs := http.FileServer(http.Dir("static/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	go scheduleWorkers()
	go scheduleRecv()

	_ = http.ListenAndServe(":80", nil)
}

type Identify struct {
	Id int `json:"id"`
	Auth string `json:"authorization"`
}

func handleWs(w http.ResponseWriter, r *http.Request) {
	conn, _ := upgrade.Upgrade(w, r, nil) // error ignored for sake of simplicity
	type_, msg, err := conn.ReadMessage()
	if err == nil {
		if type_ == 1 {
			payload := Identify{}
			err := json.Unmarshal(msg, &payload)
			if err == nil {
				
			}
		}
	}

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

type ASGIResponse struct {
	headers map[string][]string `json:"-"`
	Body    string              `json:"body"`
	Status  int                 `json:"status"`
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
	val := <- readChannel
	w.WriteHeader(val.Status) // Status code
	//header := w.Header()
	//for k, v := range val.headers {
	//	header.Add(k, v)
	//}
	_, _ = fmt.Fprint(w, val.Body) // Request Body
}
