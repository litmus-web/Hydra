package server

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"

	"../prefork"
)


///
///  Starting Areas, spawns all the servers
///
func StartServers(mainHost string, workerPort int)  {
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


	if err := fasthttp.ListenAndServe(fmt.Sprintf(":%v", workerPort), requestHandler); err != nil {
		panic(err)
	}
}

func startMainServer(mainHost string) {
	server := &fasthttp.Server{
		Handler: anyHTTPHandler,
	}
	preforkServer := prefork.New(server)
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
var cache = make(map[uint64]ASGIResponse)
var cacheLock = sync.RWMutex{}

var count uint64 = 0


///
///  General Structs for communication between systems.
///
type OutGoingRequest struct {
	RequestId uint64    			`json:"request_id"`
	Method    string 				`json:"method"`
	Remote    string				`json:"remote"`
	Path      string				`json:"path"`
	Headers   map[string]string		`json:"headers"`
	Version   string				`json:"version"`
	Body      string				`json:"body"`
	Query     string				`json:"query"`
}

type ASGIResponse struct {
	RequestId uint64	`json:"request_id"`
	Status int			`json:"status"`
	Headers [][]string	`json:"headers"`
	Body string			`json:"body"`
}


///
///  Main area where all incoming requests get sent.
///
func anyHTTPHandler(ctx *fasthttp.RequestCtx) {
	atomic.AddUint64(&count, 1)

	toGo := OutGoingRequest{
		RequestId: count,
		Method:    string(ctx.Method()),
		Remote:    ctx.RemoteAddr().String(),
		Path:      string(ctx.Path()),
		Headers:   make(map[string]string),
		Version:   "HTTP/1.1",
		Body:      "",
		Query:     ctx.QueryArgs().String(),
	}
	toPythonChan<-toGo

	start := time.Now()
	for {
		cacheLock.RLock()
		res, found := cache[count]
		cacheLock.RUnlock()
		if found {
			for _, val := range res.Headers {
				ctx.Response.Header.Set(val[0], val[1])
			}
			ctx.SetStatusCode(res.Status)
			fmt.Fprintln(ctx, res.Body)
			break
		} else {
			delta := time.Now().Sub(start)
			if delta.Seconds() > 20 {
				ctx.SetStatusCode(503)
				fmt.Fprintln(ctx, "res.Body")
				break
			}
		}
		time.Sleep(12 * time.Microsecond)
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
		cacheLock.Lock()
		cache[incoming.RequestId] = incoming
		cacheLock.Unlock()
	}
}

func handleWrite(conn *websocket.Conn) {
	for {
		toGo := <-toPythonChan
		_ = conn.WriteJSON(toGo)
	}
}