package server


type OutgoingRequest struct {
	Op		  int				`json:"op"`
	RequestId uint64            `json:"request_id"`
	Method    string            `json:"method"`
	Remote    string            `json:"remote"`
	Path      string            `json:"path"`
	Headers   string 			`json:"headers"`
	Version   string            `json:"version"`
	Body      string            `json:"body"`
	Query     string            `json:"query"`
}

type OutgoingShardPayload struct {
	Outgoing OutgoingRequest
	//RecvChannel chan IncomingResponse
}

type IncomingResponse struct {
	Op		  int		   	   `json:"op"`
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