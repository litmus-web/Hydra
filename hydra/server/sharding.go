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

type ShardManager struct {
	shards *hashmap.HashMap // map[uint64]*Shard
}

func (sm *ShardManager) AddShard(shard *Shard) {
	sm.shards.Set(shard.ShardId, shard)
}

func (sm *ShardManager) RemoveShard(shardId uint64) {
	sm.shards.Del(shardId)
}

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
