package server

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/valyala/fasthttp"

	"../prefork"
)

var (
	nextShard      chan uint64
	nextResponseId uint64 = 0

	countPool = sync.Pool{
		New: func() interface{} {
			atomic.AddUint64(&nextResponseId, 1)
			newId := nextResponseId
			return RequestPack{
				ReqId:       newId,
				ShardId:     <-nextShard,
				RecvChannel: make(chan IncomingResponse),
				ModRequest: OutgoingRequest{
					Op:        1,
					RequestId: newId,
				},
			}
		},
	}
)

func init() {
	nextShard = make(chan uint64)
	go func() {
		var totalLength uint64
		var counter uint64 = 1
		for {
			totalLength = uint64(shardManager.Shards.Len())
			if counter > totalLength {
				counter = 1
			}

			nextShard <- counter

			counter += 1
		}
	}()
}

/*
	startMainServer (public) starts the pre-forking FastHTTP server binding to the
	set address of `mainHost`
*/
func StartMainServer(mainHost string, workerCount int) {
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

func parseHeaders(ctx *fasthttp.RequestCtx) string {
	return ctx.Request.Header.String()
}

/*
	Any external request goes through here first
	all parsing, caching and checks are done then handed
	off to the workers, this is really useful if you want
	to remove load from python via caching or request blocking.
*/
func anyHTTPHandler(ctx *fasthttp.RequestCtx) {
	reqHelper := countPool.Get().(RequestPack)

	reqHelper.ModRequest.Headers = parseHeaders(ctx)
	err := recover()
	if err != nil {
		ctx.SetStatusCode(400)
		ctx.SetBodyString("Invalid request")
		countPool.Put(reqHelper)
		return
	}

	reqHelper.ModRequest.Method = string(ctx.Method())
	reqHelper.ModRequest.Remote = ctx.RemoteAddr().String()
	reqHelper.ModRequest.Path = string(ctx.Path())
	reqHelper.ModRequest.Version = "HTTP/1.1"
	reqHelper.ModRequest.Body = ""
	reqHelper.ModRequest.Query = ctx.QueryArgs().String()

	exists := shardManager.SubmitToShard(reqHelper.ShardId, &reqHelper.ModRequest, reqHelper.RecvChannel)

	if !exists {
		ctx.SetStatusCode(503)
		_, _ = fmt.Fprintf(
			ctx, "Internal Server error: Shard with Id: %v does not exist.", &reqHelper.ShardId)
		return
	}

	response := <-reqHelper.RecvChannel

	countPool.Put(reqHelper)

	ctx.SetStatusCode(response.Status)
	ctx.SetBodyString(response.Body)

	var head []string
	for _, head = range response.Headers {
		ctx.Response.Header.Set(head[0], head[1])
	}
}
