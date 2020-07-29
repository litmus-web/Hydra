package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"math/rand"
	"net/http"

	"strings"
	"time"

	"./process_management"
)

var upgrade = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randAuth(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

var workers = make(map[string]*websocket.Conn)
var sendChannel = make(chan map[string]interface{})
var readChannel = make(chan ASGIResponse)
var authorization = randAuth(12)

func main() {
	target := process_management.TargetApp{
		Target: "tests\\worker_test\\process_client.py",
		App:    "app",
		Auth:   authorization,
	}
	fmt.Println(target)
	go process_management.StartWorkers(2, target)

	http.HandleFunc("/", requestHandler)
	http.HandleFunc("/workers", handleWs)

	fs := http.FileServer(http.Dir("static/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	go scheduleWorkers()
	go scheduleRecv()

	_ = http.ListenAndServe(":80", nil)
}

type Identify struct {
	Id_  string `json:"id_"`
	Auth string `json:"authorization"`
}

func handleWs(w http.ResponseWriter, r *http.Request) {
	conn, _ := upgrade.Upgrade(w, r, nil) // error ignored for sake of simplicity
	remote := strings.SplitAfter(conn.RemoteAddr().String(), ":")[0][:9]
	if remote != "127.0.0.1" {
		_ = conn.Close()
		return
	}
	_ = conn.WriteMessage(1, []byte("WORKER.IDENTIFY"))
	type_, msg, err := conn.ReadMessage()
	if err == nil {
		if type_ == 1 {
			payload := Identify{}
			err := json.Unmarshal(msg, &payload)
			if err == nil {
				if payload.Auth == authorization {
					log.Println("Worker", payload.Id_, "successfully connected to server.")
					workers[payload.Id_] = conn
				} else {
					_ = conn.Close()
					return
				}
			}
		}
	}

	workers["abc"] = conn
}

func scheduleWorkers() {
	for {
		if workers["1"] != nil {
			msg := <-sendChannel
			_ = workers["1"].WriteJSON(msg)
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
		if workers["1"] != nil {
			_, msg, err := workers["1"].ReadMessage()
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
	w.WriteHeader(val.Status) // Status code
	//header := w.Header()
	//for k, v := range val.headers {
	//	header.Add(k, v)
	//}
	_, _ = fmt.Fprint(w, val.Body) // Request Body
}
