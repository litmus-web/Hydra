package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/valyala/fasthttp"

	"./prefork"
)

// Flags
var (
	target = flag.String(
		"filepath",
		"./client.py",
		"Specify the file containing the app")
	app = flag.String(
		"app",
		"app",
		"Specify the app variable/object")
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var recvChannel chan map[string]interface{} = nil
var cache cacheMap

var ops int64
var count = atomic.AddInt64(&ops, 1)

type cacheMap struct {
	internal map[int]ASGIResponse
	lock     sync.RWMutex
}

func (m *cacheMap) Get(key int) (ASGIResponse, bool) {
	m.lock.RLock()
	v, ok := m.internal[key]
	m.lock.RUnlock()
	return v, ok
}
func (m *cacheMap) Store(key int, val ASGIResponse) {
	m.lock.Lock()
	m.internal[key] = val
	m.lock.Unlock()
}

type ASGIResponse struct {
	RequestId int                 `json:"request_id"`
	headers   map[string][]string `json:"-"`
	Body      string              `json:"body"`
	Status    int                 `json:"status"`
}

func main() {
	flag.Parse()

	recvChannel = make(chan map[string]interface{})
	cache = cacheMap{
		internal: make(map[int]ASGIResponse),
		lock:     sync.RWMutex{},
	}

	go handleWrite()
	go handleRead()

	// Start Go server
	startServer()
}

func startServer() {
	server := &fasthttp.Server{
		Handler: requestHandler,
		Name:    "Sandman",
	}

	preforkServer := prefork.New(server, true)

	if err := preforkServer.ListenAndServe(":8000"); err != nil {
		log.Fatal(err)
	}
}

func handleWrite() {
	for {
		outgoing := <-recvChannel
		msg, err := json.Marshal(outgoing)
		if err == nil {
			_, err = fmt.Printf("%v\n", string(msg))
			if err != nil {
				log.Fatal("Message send failed.")
			}
		}

	}
}

func handleRead() {
	r := bufio.NewReader(os.Stdin)
	for {
		out, _ := r.ReadBytes('\n')
		out = bytes.TrimRight(out, "\n")

		payload := ASGIResponse{}
		err := json.Unmarshal(out, &payload)
		if err == nil {
			cache.Store(payload.RequestId, payload)
		} else {
			log.Printf("Logging from worker: %v", string(out))
		}
	}
}

func requestHandler(ctx *fasthttp.RequestCtx) {
	atomic.AddInt64(&count, 1)
	key := int(count)
	toGo := map[string]interface{}{
		"request_id": key,
		"method":     string(ctx.Method()),
		"path":       string(ctx.Path()),
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
		time.Sleep(10 * time.Microsecond)
	}
}
