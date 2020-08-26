package server

import (
	"fmt"
	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
	"log"
	"sync"
	"sync/atomic"

	"../prefork"
)

///
///  Starting Areas, spawns all the servers
///
func StartServers(mainHost string, workerPort int) {
	go startWorkerServer(workerPort)
	startMainServer(mainHost)
}

func startWorkerServer(workerPort int) {
	requestHandler := func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case "/workers":
			workerHandler(ctx)
		default:
			ctx.Error("Unsupported path", fasthttp.StatusNotFound)
		}
	}

	binding := fmt.Sprintf("127.0.0.1:%v", workerPort)
	if err := fasthttp.ListenAndServe(binding, requestHandler); err != nil {
		panic(err)
	}
}

func startMainServer(mainHost string) {
	server := &fasthttp.Server{
		Handler: anyHTTPHandler,
	}

	preforkServer := prefork.New(server)

	if !prefork.IsChild() {
		fmt.Printf("Server started server on http://%s\n", mainHost)
	}

	if err := preforkServer.ListenAndServe(mainHost); err != nil {
		panic(err)
	}
}

///
///  General variables and constants for communication between systems.
///
var upgrader = websocket.FastHTTPUpgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var toPythonChan = make(chan OutGoingRequest)
var cache = make(map[uint64]chan ASGIResponse)
var cacheLock = sync.RWMutex{}

var count uint64 = 0
var countPool = sync.Pool{
	New: func() interface{} {
		atomic.AddUint64(&count, 1)
		return count
	},
}

///
///  General Structs for communication between systems.
///
type OutGoingRequest struct {
	RequestId uint64            `json:"request_id"`
	Method    string            `json:"method"`
	Remote    string            `json:"remote"`
	Path      string            `json:"path"`
	Headers   map[string]string `json:"headers"`
	Version   string            `json:"version"`
	Body      string            `json:"body"`
	Query     string            `json:"query"`
}

type ASGIResponse struct {
	Meta      IncomingMetadata `json:"meta_data"`
	RequestId uint64           `json:"request_id"`
	Type      string           `json:"type"`
	Status    int              `json:"status"`
	Headers   [][]string       `json:"headers"`
	Body      string           `json:"body"`
	MoreBody  bool             `json:"more_body"`
}

type IncomingMetadata struct {
	ResponseType string `json:"meta_response_type"`
}

///
///  Main area where all incoming requests get sent.
///
func anyHTTPHandler(ctx *fasthttp.RequestCtx) {

	reqId := countPool.Get().(uint64)

	toGo := OutGoingRequest{
		RequestId: reqId,
		Method:    string(ctx.Method()),
		Remote:    ctx.RemoteAddr().String(),
		Path:      string(ctx.Path()),
		Headers:   make(map[string]string),
		Version:   "HTTP/1.1",
		Body:      "",
		Query:     ctx.QueryArgs().String(),
	}
	toPythonChan <- toGo

	incomingChan := make(chan ASGIResponse)

	cacheLock.Lock()
	cache[reqId] = incomingChan
	cacheLock.Unlock()

	workerResponse := <-incomingChan

	switch workerResponse.Meta.ResponseType {

	case "timeout":
		writeTimeout(ctx)
		countPool.Put(reqId)
		return

	case "partial":
		invokePartial(ctx, workerResponse, incomingChan, reqId)

	case "complete":
		invokeAll(ctx, workerResponse)

	default:
		log.Printf("Invalid response type recieved from worker with Id: %v\n", reqId)
		return
	}
}

///
///  This is the worker area, responsible for upgrading the WS connection
///  to the server allowing for fast transactions between processes.
///
func workerHandler(ctx *fasthttp.RequestCtx) {
	_ = upgrader.Upgrade(ctx, upgradedWebsocket)
}

func upgradedWebsocket(conn *websocket.Conn) {
	go handleRead(conn)
	handleWrite(conn)
}

func handleRead(conn *websocket.Conn) {
	for {
		incoming := ASGIResponse{}
		err := conn.ReadJSON(&incoming)
		if err != nil {
			log.Fatal(err)
		}
		cacheLock.RLock()
		cache[incoming.RequestId] <- incoming
		cacheLock.RUnlock()
	}
}

func handleWrite(conn *websocket.Conn) {
	for {
		toGo := <-toPythonChan
		_ = conn.WriteJSON(toGo)
	}
}
