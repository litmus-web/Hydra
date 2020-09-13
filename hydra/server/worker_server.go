package server

import (
	"fmt"
	"sync/atomic"

	"github.com/cornelk/hashmap"
	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"

	"../process_manager"
)

var (
	nextShardId uint64 = 0

	upgrader = websocket.FastHTTPUpgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

/*
	startWorkerServer is internal server that is reserved just for worker
	processes, and the only entry point is via `ws://127.0.0.1:workerPort/workers`
	anything else is ignored and returns a 403 or method not allowed.

	Invokes:
		- workerHandler()
*/
func StartWorkerServer(workerManager process_manager.ExternalWorkers) error {

	requestHandler := func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case "/workers":
			workerHandler(ctx, workerManager.WorkerAuth)
		default:
			ctx.Error("Unsupported path", fasthttp.StatusNotFound)
		}
	}

	ended := make(chan error)

	//go func() {
	//		workerManager.StartExternalWorkers()
	//		ended <- nil
	//}()

	go func() {
		binding := fmt.Sprintf("127.0.0.1:%v", workerManager.ConnectionPort)
		fmt.Println("Binding to: ", binding)
		err := fasthttp.ListenAndServe(binding, requestHandler)
		ended <- err
	}()

	return <-ended
}

func workerHandler(ctx *fasthttp.RequestCtx, auth string) {
	reqAuth := string(ctx.Request.Header.Peek("Authorization"))
	if reqAuth != auth {
		ctx.SetStatusCode(403)
		ctx.SetBodyString("Not authorized")
		return
	}

	_ = upgrader.Upgrade(ctx, upgradedWebsocket)
}

func upgradedWebsocket(conn *websocket.Conn) {
	atomic.AddUint64(&nextShardId, 1)

	shard := Shard{
		ShardId:         nextShardId,
		OutgoingChannel: make(chan *OutgoingRequest),
		RecvCache:       &hashmap.HashMap{},
		conn:            conn,
	}

	shardManager.AddShard(&shard)

	shard.Start()
}
