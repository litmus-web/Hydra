package server

import (
	"fmt"
	"log"
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

func StartWorkerServer(workerPort int, workerManager process_manager.ExternalWorkers) {
	/*
		startWorkerServer is internal server that is reserved just for worker
		processes, and the only entry point is via `ws://127.0.0.1:workerPort/workers`
		anything else is ignored and returns a 403 or method not allowed.

		Invokes:
			- workerHandler()
	*/
	requestHandler := func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case "/workers":
			workerHandler(ctx)
		default:
			ctx.Error("Unsupported path", fasthttp.StatusNotFound)
		}
	}

	ended := make(chan error)

	go func() {
		workerManager.StartExternalWorkers()
		ended <- nil
	}()

	go func() {
		binding := fmt.Sprintf("127.0.0.1:%v", workerPort)
		fmt.Println("Binding to: ", binding)
		err := fasthttp.ListenAndServe(binding, requestHandler)
		ended <- err
	}()

	err := <-ended
	if err != nil {
		log.Fatalln(err)
	}
}

func workerHandler(ctx *fasthttp.RequestCtx) {
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
