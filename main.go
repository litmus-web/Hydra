package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/fasthttp-contrib/websocket"
	"github.com/valyala/fasthttp"

	"./process_management"
)

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
var mapMutex = sync.Mutex{}

func main() {
	target := process_management.TargetApp{
		Target: "tests\\worker_test\\process_client.py",
		App:    "app",
		Auth:   authorization,
	}
	fmt.Println(target)
	go process_management.StartWorkers(2, target)

	requestHandler := func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case "/workers":
			onWsConnect(ctx)
		default:
			requestHandler(ctx)
		}
	}

	_ = fasthttp.ListenAndServe(":80", requestHandler)
}

type Identify struct {
	Id_  string `json:"id_"`
	Auth string `json:"authorization"`
}

type ASGIResponse struct {
	headers map[string][]string `json:"-"`
	Body    string              `json:"body"`
	Status  int                 `json:"status"`
}

var upgrader = websocket.New(handleWs)

func onWsConnect(ctx *fasthttp.RequestCtx) {
	_ = upgrader.Upgrade(ctx)
}

func handleWs(conn *websocket.Conn) {
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
					go handleWriteWebSocketConn(conn)
					handleReadWebSocketConn(conn)
				} else {
					_ = conn.Close()
					return
				}
			}
		}
	}
}

func handleReadWebSocketConn(conn *websocket.Conn) {
	for {
		type_, msg, err := conn.ReadMessage()
		if err == nil {
			if type_ == 1 {
				payload := ASGIResponse{}
				err := json.Unmarshal(msg, &payload)
				if err == nil {
					readChannel <- payload
				}
			}
		}
	}
}

func handleWriteWebSocketConn(conn *websocket.Conn) {
	for {
		outgoing := <-sendChannel
		msg, err := json.Marshal(outgoing)
		if err == nil {
			err = conn.WriteMessage(1, msg)
			if err != nil {
				log.Fatal("Message send failed.")
			}
		}
	}
}

func requestHandler(ctx *fasthttp.RequestCtx) {
	toGo := map[string]interface{}{
		"method":  ctx.Method(),
		"path":    ctx.Path(),
		"remote":  ctx.RemoteAddr().String(),
		"headers": ctx.Request.Header,
		"body":    string(ctx.Request.Body()),
		"server":  "Sandman",
	}

	sendChannel <- toGo
	response := <-readChannel
	ctx.SetStatusCode(response.Status)

	//header := w.Header()
	//for k, v := range val.headers {
	//	header.Add(k, v)
	//}
	_, _ = fmt.Fprint(ctx, response.Body) // Request Body
}
