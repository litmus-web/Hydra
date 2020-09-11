package server

import (
	"github.com/cornelk/hashmap"
	"github.com/fasthttp/websocket"
	"log"
)

var (
	shardManager ShardManager
)

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

// Used to represent a ws connection in itself
type Shard struct {
	ShardId uint64

	OutgoingChannel chan *OutgoingRequest

	RecvCache *hashmap.HashMap

	conn *websocket.Conn
}

func (s *Shard) SetConn(conn *websocket.Conn) {
	s.conn = conn
}

func (s *Shard) SubmitRequest(request *OutgoingRequest, recv chan IncomingResponse) {
	s.OutgoingChannel <- request
	s.RecvCache.GetOrInsert(request.RequestId, recv)
}

func (s *Shard) Start() {
	go s.handleRead()
	s.handleWrite()
}

func (s *Shard) handleWrite() {
	var outgoing *OutgoingRequest
	for outgoing = range s.OutgoingChannel {
		_ = s.conn.WriteJSON(outgoing)
	}
}

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
