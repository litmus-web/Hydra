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

//
//  Starting Areas, spawns all the servers
//

/*
	StartServers is invokes both the main server and the worker server.

	Invokes:
		- go startWorkerServer()
		- startMainServer()
*/
func StartServers(mainHost string, workerPort int) {
	go startWorkerServer(workerPort)
	startMainServer(mainHost)
}

/*
	startWorkerServer is internal server that is reserved just for worker
	processes, and the only entry point is via `ws://127.0.0.1:workerPort/workers`
	anything else is ignored and returns a 403 or method not allowed.

	Invokes:
		- workerHandler()
*/
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

/*
	startMainServer (public) starts the pre-forking FastHTTP server binding to the
	set address of `mainHost`
*/
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

//
//  General variables and constants for communication between systems.
//
var upgrader = websocket.FastHTTPUpgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var count uint64 = 0
var countPool = sync.Pool{
	New: func() interface{} {
		atomic.AddUint64(&count, 1)
		return count
	},
}

//
//  General Structs for communication between systems.
//
type OutGoingRequest struct {
	OpCode    int               `json:"op"`
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

type ShardIdentify struct {
	ShardId string `json:"shard_id"`
}

///
///  Main area where all incoming requests get sent.
///

/*
	anyHTTPHandler is the main handler that handles all incoming
	http requests on the main entry point, from `/` to `/abc/123`

	A request id is taken from the pool to make use of recycling objects
	if there is no free ids one is atomically gotten which is then later
	returned.

	It then creates a `OutGoingRequest` interface containing anything needed
	for the ASGI, WSGI or Raw systems. todo add headers instead of a empty map
	Because FastHTTP does not support http/2 yet we can hardcode the request
	version.

	A channel is made and added to the cache waiting for the responses of the
	workers, the server then waits for the responses coming from the channel.

	Upon a incoming response it matches the metadata type (timeout, partial, complete)
	to check if the worker has timed out internally, has a one shot response or a multi-part
	response.

	Invokes:
		- writeTimeout()	`timeout`
		- invokePartial()	`partial`
		- invokeAll()		`complete`

*/
func anyHTTPHandler(ctx *fasthttp.RequestCtx) {

	reqId := countPool.Get().(uint64)

	toGo := OutGoingRequest{
		OpCode:    1,
		RequestId: reqId,
		Method:    string(ctx.Method()),
		Remote:    ctx.RemoteAddr().String(),
		Path:      string(ctx.Path()),
		Headers:   make(map[string]string),
		Version:   "HTTP/1.1",
		Body:      "",
		Query:     ctx.QueryArgs().String(),
	}
	pair := acquireShard("1")
	pair.In <- toGo

	var workerResponse ASGIResponse
	for workerResponse = range pair.Out {
		if workerResponse.RequestId == reqId {
			break
		}
	}

	switch workerResponse.Meta.ResponseType {

	case "timeout":
		writeTimeout(ctx)
		countPool.Put(reqId)
		return

	case "partial":
		invokePartial(ctx, workerResponse, pair.Out, reqId)

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

/*
	upgradedWebsocket is the callback after a websocket connection
	has been successfully upgraded and starts the read goroutine and
	then handles any writing.

	Invokes:
		- go handleRead(conn)
		- handleWrite(conn)
*/
func upgradedWebsocket(conn *websocket.Conn) {
	callback := make(chan ShardIdentify)
	go handleRead(conn, callback)
	handleWrite(conn, callback)
}

/*
	handleRead is a infinite loop waiting for incoming
	websocket messages to marshal to a ASGIResponse
	which is the sent via a channel through a RwLock.
*/
func handleRead(conn *websocket.Conn, c chan ShardIdentify) {
	// Send ident payload only send
	v := map[string]int{
		"op": 0,
	}
	_ = conn.WriteJSON(v)

	// Read returned identify
	s := ShardIdentify{}
	err := conn.ReadJSON(&s)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(s)
	c <- s

	pair := acquireShard(s.ShardId)
	for {
		incoming := ASGIResponse{}
		err := conn.ReadJSON(&incoming)
		if err != nil {
			log.Fatal(err)
		}

		pair.Out <- incoming
	}
}

/*
	handleWrite is just a infinite loop sending anything coming
	through the channel to the websocket worker.
*/
func handleWrite(conn *websocket.Conn, c chan ShardIdentify) {
	shard := <-c

	pair := ChannelPair{
		In:  make(chan OutGoingRequest),
		Out: make(chan ASGIResponse),
	}
	setShard(shard.ShardId, pair)

	var toGo OutGoingRequest
	for toGo = range pair.In {
		_ = conn.WriteJSON(toGo)
	}
}
