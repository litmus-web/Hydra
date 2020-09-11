package server

import (
	"fmt"
	"sync/atomic"

	"github.com/cornelk/hashmap"
	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
)

var (
	nextShardId uint64 = 0

	upgrader = websocket.FastHTTPUpgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

func StartWorkerServer(workerPort int) {
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

	binding := fmt.Sprintf("127.0.0.1:%v", workerPort)
	fmt.Println("Binding to: ", binding)
	if err := fasthttp.ListenAndServe(binding, requestHandler); err != nil {
		panic(err)
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
