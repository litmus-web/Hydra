package server

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/valyala/fasthttp"

	"../prefork"
)

var (
	nextResponseId uint64 = 0
    countPool = sync.Pool{
		New: func() interface{} {
			atomic.AddUint64(&nextResponseId, 1)
			return nextResponseId
		},
	}
)



func StartMainServer(mainHost string, workerCount int) {
	/*
		startMainServer (public) starts the pre-forking FastHTTP server binding to the
		set address of `mainHost`
	*/
	server := &fasthttp.Server{
		Handler: anyHTTPHandler,
	}

	preforkServer := prefork.New(server, workerCount)

	if !prefork.IsChild() {
		fmt.Printf("Server started server on http://%s\n", mainHost)
	}

	if err := preforkServer.ListenAndServe(mainHost); err != nil {
		panic(err)
	}
}

func anyHTTPHandler(ctx *fasthttp.RequestCtx) {
	myId := countPool.Get().(uint64)

	toGo := OutgoingShardPayload {
		OutgoingRequest{
			Op: 1,
			RequestId: myId,
			Method:    string(ctx.Method()),
			Remote:    ctx.RemoteAddr().String(),
			Path:      string(ctx.Path()),
			Headers:   ctx.Request.Header.String(),
			Version:   "HTTP/1.1",
			Body:      "",
			Query:     ctx.QueryArgs().String(),
		},
	}

	var shardId uint64
	shardId = 1

	myChannel := make(chan  IncomingResponse)
	exists := shardManager.SubmitToShard(shardId, toGo, myChannel)

	if !exists {
		ctx.SetStatusCode(503)
		_, _ = fmt.Fprintf(ctx, "Internal Server error: Shard with Id: %v does not exist.", shardId)
		return
	}
	_ =<-myChannel
	countPool.Put(myId)

	ctx.SetStatusCode(200)
	ctx.SetBody([]byte("Hello World"))
	//var out IncomingResponse
	//for out = range recv {
	//	fmt.Println(out)
	//}
}
