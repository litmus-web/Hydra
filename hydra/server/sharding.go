package server

import (
	"github.com/cornelk/hashmap"
	"github.com/fasthttp/websocket"
	"github.com/valyala/fastjson"
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

	var p fastjson.Parser
	var msg []byte
	var v *fastjson.Value

	incoming := IncomingResponse{}

	for {
		_, msg, err = s.conn.ReadMessage()
		if err != nil {
			log.Fatal(err)
		}

		v, err = p.ParseBytes(msg)
		if err != nil {
			log.Fatal(err)
		}
		parseMsg(v, &incoming)

		channel, ok = s.RecvCache.Get(incoming.RequestId)
		if ok {
			cha = (channel).(chan IncomingResponse)
			cha <- incoming
		}
	}
}

func parseMsg(v *fastjson.Value, ir *IncomingResponse) {
	ir.Op = v.GetInt("op")
	ir.RequestId = v.GetUint64("request_id")

	ir.Meta = v.GetStringBytes("meta_data")
	ir.Type = v.GetStringBytes("type")
	ir.Status = v.GetInt("status")

	ir.Body = v.GetStringBytes("body")
	ir.MoreBody = v.GetBool("more_body")

	ir.Headers = v.GetArray("headers")
}
