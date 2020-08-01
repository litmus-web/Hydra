package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/prefork"

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

var recvChannel chan map[string]interface{} = nil
var cache cacheMap
var authorization = ""
var ops int64
var count = atomic.AddInt64(&ops, 1)

type workers struct {
	workers map[int]cacheMap
	lock    sync.Mutex
}

func (w *workers) Get(key int) (cacheMap, bool) {
	w.lock.Lock()
	v, ok := w.workers[key]
	w.lock.Unlock()
	return v, ok
}
func (w *workers) Store(key int, val cacheMap) {
	w.lock.Lock()
	w.workers[key] = val
	w.lock.Unlock()
}

type cacheMap struct {
	internal map[int]ASGIResponse
	lock     sync.Mutex
}

func (m *cacheMap) Get(key int) (ASGIResponse, bool) {
	m.lock.Lock()
	v, ok := m.internal[key]
	m.lock.Unlock()
	return v, ok
}

func (m *cacheMap) Store(key int, val ASGIResponse) {
	m.lock.Lock()
	m.internal[key] = val
	m.lock.Unlock()
}

// Flags
var (
	target = flag.String(
		"filepath",
		"tests\\worker_test\\client.py",
		"Specify the file containing the app")
	app = flag.String(
		"app",
		"app",
		"Specify the app variable/object")
)

func main() {
	// Setting vars
	authorization = randAuth(12)
	recvChannel = make(chan map[string]interface{})
	cache = cacheMap{
		internal: make(map[int]ASGIResponse),
		lock:     sync.Mutex{},
	}

	flag.Parse()
	target := process_management.TargetApp{
		Target: *target,
		App:    *app,
		Auth:   authorization,
	}
	fmt.Println(target)
	go process_management.StartWorkers(1, target) // Do not change, archive of old method

	requestHandler := func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case "/workers":
			onWsConnect(ctx)
		default:
			requestHandler(ctx)
		}
	}

	if runtime.GOOS == "windows" {
		if prefork.IsChild() {
			log.Println("Pre-Forking is not supported on windows runtime, Switching to standard mode.")
		}
		if err := fasthttp.ListenAndServe(":80", requestHandler); err != nil {
			log.Fatal(err)
		}
	} else {
		if prefork.IsChild() {
			log.Println("Pre-Forking mode enabled.")
		}
		server := &fasthttp.Server{
			Handler: requestHandler,
			Name:    "Sandman",
		}

		preforkServer := prefork.New(server)

		if err := preforkServer.ListenAndServe(":80"); err != nil {
			log.Fatal(err)
		}
	}
}

type Identify struct {
	Id_  string `json:"id_"`
	Auth string `json:"authorization"`
}

type ASGIResponse struct {
	RequestId int                 `json:"request_id"`
	headers   map[string][]string `json:"-"`
	Body      string              `json:"body"`
	Status    int                 `json:"status"`
}

var upgrader = websocket.FastHTTPUpgrader{}

func onWsConnect(ctx *fasthttp.RequestCtx) {
	_ = upgrader.Upgrade(ctx, handleWs)
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

func handleWriteWebSocketConn(conn *websocket.Conn) {
	for {
		outgoing := <-recvChannel
		msg, err := json.Marshal(outgoing)
		if err == nil {
			err = conn.WriteMessage(1, msg)
			if err != nil {
				log.Fatal("Message send failed.")
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
					cache.Store(payload.RequestId, payload)
				}
			}
		}
	}
}

func requestHandler(ctx *fasthttp.RequestCtx) {
	atomic.AddInt64(&count, 1)
	key := int(count)
	toGo := map[string]interface{}{
		"request_id": key,
		"method":     ctx.Method(),
		"path":       ctx.Path(),
		"remote":     ctx.RemoteAddr().String(),
		"headers":    ctx.Request.Header,
		"body":       string(ctx.Request.Body()),
		"server":     "Sandman",
	}
	recvChannel <- toGo

	for {
		v, ok := cache.Get(key)
		if ok {
			ctx.SetStatusCode(v.Status)
			//header := w.Header()
			//for k, v := range val.headers {
			//	header.Add(k, v)
			//}
			_, _ = fmt.Fprint(ctx, v.Body) // Request Body
			return
		}
		time.Sleep(5 * time.Millisecond)
	}

}
