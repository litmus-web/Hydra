package server

import (
	"github.com/fasthttp/websocket"
	"log"
	"sync"
)

var (
	shardManager ShardManager
)

func init() {
	shardManager = ShardManager{
		shards:     make(map[uint64]*Shard),
		shardsLock: sync.RWMutex{},
	}
}

type ShardManager struct {
	shards     map[uint64]*Shard
	shardsLock sync.RWMutex
}

func (sm *ShardManager) AddShard(shard *Shard) {
	sm.shardsLock.Lock()
	defer sm.shardsLock.Unlock()
	sm.shards[shard.ShardId] = shard
}

func (sm *ShardManager) RemoveShard(shardId uint64) {
	sm.shardsLock.Lock()
	defer sm.shardsLock.Unlock()
	delete(sm.shards, shardId)
}

func (sm *ShardManager) SubmitToShard(shardId uint64, out OutgoingShardPayload, recv chan IncomingResponse) bool {
	sm.shardsLock.RLock()
	s := sm.shards[shardId]
	if s == nil {
		sm.shardsLock.RUnlock()
		return false
	}
	s.SubmitRequest(out, recv)

	sm.shardsLock.RUnlock()
	return true
}

// Used to represent a ws connection in itself
type Shard struct {
	ShardId uint64

	OutgoingChannel chan OutgoingRequest


	RecvCache map[uint64]chan IncomingResponse
	RecvLock sync.Mutex

	conn *websocket.Conn
}

func (s *Shard) SetConn(conn *websocket.Conn) {
	s.conn = conn
}

func (s *Shard) SubmitRequest(request OutgoingShardPayload, recv chan IncomingResponse) {
	s.OutgoingChannel <- request.Outgoing

	s.RecvLock.Lock()
	s.RecvCache[request.Outgoing.RequestId] = recv
	s.RecvLock.Unlock()
}

func (s *Shard) Start() {
	go s.handleRead()
	s.handleWrite()
}

func (s *Shard) handleWrite()   {
	var outgoing OutgoingRequest
	for outgoing = range s.OutgoingChannel {
		_ = s.conn.WriteJSON(outgoing)
	}
}

func (s *Shard) handleRead()   {
	var incoming IncomingResponse
	var err error
	var chann chan IncomingResponse
	var ok bool

	for {
		incoming = IncomingResponse{}
		err = s.conn.ReadJSON(&incoming)
		if err != nil {
			log.Fatal(err)
		}

		s.RecvLock.Lock()
		if chann, ok = s.RecvCache[incoming.RequestId]; ok {
			chann <- incoming
			// delete(s.RecvCache, incoming.RequestId)
		}
		s.RecvLock.Unlock()
	}
}

