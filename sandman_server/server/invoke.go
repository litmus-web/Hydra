package server

import (
	"fmt"
	"log"

	"github.com/valyala/fasthttp"
)

/*
	writeTimeout represents a worker just timing out or something just
	breaking everything. Basically a "EVERYONE PANIC!" response that should
	NEVER be triggered, by default the timeout is 15 seconds before Sandman
	will automatically stop the worker request returning a 503 status code to
	the incoming external request, the request id is *not* returned to the pool
	to stop the workers possibly returning someone else's response in another request.

	For all intensive purposes the request is dead and dangerous.
*/
func writeTimeout(ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(503)
	ctx.SetBody([]byte("Sandman Automated Timeout: Worker took to long to respond."))
}

/*
	invokePartial anticipates the response from	the websocket coming in
	parts, it expects a `response.start` incoming type first followed by
	`response.body` types as it writes these to the	socket individually.

	This is used by anything where the body of the request will exceed the
	maximum message length on the WS where overhead would be higher than
	doing it in parts.
	The function will finish when the response gives a 	more_body = False
	response, returning the request id to the pool.

	Invokes:
		- writeStart() x 1
		- writeBody() x n
*/
func invokePartial(
	ctx *fasthttp.RequestCtx,
	workerResponse ASGIResponse,
	incomingChan chan ASGIResponse,
	reqId uint64,
) {
	switch workerResponse.Type {
	case "response.start":
		writeStart(ctx, workerResponse)
	case "response.body":
		writeBody(ctx, workerResponse)
	default:
		log.Fatalln("well fuck")
	}

	for workerResponse := range incomingChan {
		if workerResponse.RequestId == reqId {
			switch workerResponse.Type {
			case "response.start":
				writeStart(ctx, workerResponse)
			case "response.body":
				writeBody(ctx, workerResponse)
				if !workerResponse.MoreBody {
					return
				}
			default:
				log.Fatalln("well fuck")
			}
		}
	}
}

/*
	invokeAll represents a single ws response that contains
	both the status code, headers and body all in one, used
	for short and simple responses where doing it in parts
	adds more overhead than sending the body itself.
	Obviously this cannot be done for larger bodies because
	it exceeds the websocket max body limit
	(Changeable but not recommended.)

	Invokes:
		- writeStart() x 1
		- writeBody() x 1

*/
func invokeAll(
	ctx *fasthttp.RequestCtx,
	workerResponse ASGIResponse,
) {
	writeStart(ctx, workerResponse)
	writeBody(ctx, workerResponse)
}

/*
writeStart sets the status code and headers to the request
from the response, this is the equivalent of http.response.start
in a ASGI system.
*/
func writeStart(ctx *fasthttp.RequestCtx, resp ASGIResponse) {

	ctx.SetStatusCode(resp.Status)
	for _, val := range resp.Headers {
		ctx.Response.Header.Set(val[0], val[1])
	}
}

// writeBody does what it says on the tin, writes the body.
func writeBody(ctx *fasthttp.RequestCtx, resp ASGIResponse) {

	// todo add a better handler in case the external connection is closed.

	_, err := fmt.Fprintln(ctx, resp.Body)
	if err != nil {
		log.Fatal(err)
	}
}
