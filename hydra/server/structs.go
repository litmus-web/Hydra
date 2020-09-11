package server

type OutgoingRequest struct {
	Op        int    `json:"op"`
	RequestId uint64 `json:"request_id"`
	Method    string `json:"method"`
	Remote    string `json:"remote"`
	Path      string `json:"path"`
	Headers   string `json:"headers"`
	Version   string `json:"version"`
	Body      string `json:"body"`
	Query     string `json:"query"`
}

type IncomingResponse struct {
	Op        int              `json:"op"`
	Meta      IncomingMetadata `json:"meta_data"`
	RequestId uint64           `json:"request_id"`
	Type      string           `json:"type"`
	Status    int              `json:"status"`
	Headers   [][]string       `json:"headers"`
	Body      string           `json:"body"`
	MoreBody  bool             `json:"more_body"`
}

/*
Represents internal metadata, used for optimising
requests, small bodies aren't work sending as a separate
thing so instead we just send it in one go.

This can be represented by the following:

- partial: signals the the request is in chunks
- complete: signals that all content is in the one request.
*/
type IncomingMetadata struct {
	ResponseType string `json:"meta_response_type"`
}

/*
RequestPack acts like a zip up of required vars
like Request id, receiver channel, outgoing request.


This is used heavily for recycling variables to reduce
the load on the gc to aid performance, every little helps.
*/
type RequestPack struct {
	ReqId       uint64
	RecvChannel chan IncomingResponse
	ModRequest  OutgoingRequest
}
