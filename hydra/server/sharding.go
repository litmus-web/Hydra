package server

import (
	"github.com/cornelk/hashmap"
	"github.com/fasthttp/websocket"
	"log"
)

var shardManager ShardManager

func init() {
	shardManager = ShardManager{
		shards: &hashmap.HashMap{},
	}
}

/*
	The master controller per proc that controls all
	shard related IO and control.
*/
type ShardManager struct {
	shards *hashmap.HashMap // map[uint64]*Shard
}

/*
	Used to add a shard type to the manager, allowing for easy access and control
	later on down the line of development.
*/
func (sm *ShardManager) AddShard(shard *Shard) {
	sm.shards.Set(shard.ShardId, shard)
}

/*
	Used to remove a shard type of a given id from the manager and
	therefore removing it from the web server's usage.
*/
func (sm *ShardManager) RemoveShard(shardId uint64) {
	sm.shards.Del(shardId)
}

/*
	Use this for submitting requests to the server, it handles getting the shard
	from the hashmap and then submitting it to the shard, returning a bool to
	signal if the shard exists and has been sent the data or not.
*/
func (sm *ShardManager) SubmitToShard(shardId uint64, out *OutgoingRequest, recv chan IncomingResponse) bool {
	s, ok := sm.shards.Get(shardId)
	if !ok {
		return false
	}
	shard := (s).(*Shard)
	shard.SubmitRequest(out, recv)
	return true
}

/*
	This struct represents a individual shard, containing it's shard id,
	any applicable locks or in this cache thread safe maps, websocket connection
	and sending channel. All interactions to the websocket and shard should go
	through here.
*/
type Shard struct {
	ShardId uint64

	OutgoingChannel chan *OutgoingRequest

	RecvCache *hashmap.HashMap

	conn *websocket.Conn
}

/*
	Sets the private conn variable as a `*websocket.Conn` type.
*/
func (s *Shard) SetConn(conn *websocket.Conn) {
	s.conn = conn
}

/*
	takes a given request and a channel, sends the request to the WS handler
	channel (`OutgoingChannel`) and then Inserts the recv channel if it doesnt
	already	exist.
*/
func (s *Shard) SubmitRequest(request *OutgoingRequest, recv chan IncomingResponse) {
	s.OutgoingChannel <- request
	s.RecvCache.GetOrInsert(request.RequestId, recv)
}

/*
	A simple function that starts a thread and then handles writes
	blocking the current goroutine, this keep all lifetimes in check.
*/
func (s *Shard) Start() {
	go s.handleRead()
	s.handleWrite()
}

/*
	handles sending anything to the websocket,
	todo add better checks and events.
*/
func (s *Shard) handleWrite() {
	var outgoing *OutgoingRequest
	for outgoing = range s.OutgoingChannel {
		_ = s.conn.WriteJSON(outgoing)
	}
}

/*
this function handles reading from the shard ws connection
we define our variables before the infinite loop to maximise
performance via variable recycling rather than recreating it over
and over again.

We use the thread safe `hashmap.HashMap` type optimised for reads
because of the recycling we will be reading a lot more than inserting
when the server is running normally.
*/
func (s *Shard) handleRead() {
	var err error
	var ok bool
	var channel interface{}
	var cha chan IncomingResponse

	incoming := IncomingResponse{}

	for {
		err = s.conn.ReadJSON(&incoming)
		if err != nil {
			log.Fatal(err)
		}

		channel, ok = s.RecvCache.Get(incoming.RequestId)
		if ok {
			cha = (channel).(chan IncomingResponse)
			cha <- incoming
		}
	}
}
