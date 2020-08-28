package server

import (
	"github.com/sasha-s/go-deadlock"
)

var (
	shardChannels map[string]chan OutGoingRequest
	shardLock     deadlock.RWMutex
)

func init() {
	shardChannels = make(map[string]chan OutGoingRequest)
	shardLock = deadlock.RWMutex{}
}

func acquireShard(shardId string) chan OutGoingRequest {
	shardLock.RLock()
	temp := shardChannels[shardId]
	shardLock.RUnlock()
	return temp
}

func setShard(shardId string, recv chan OutGoingRequest) {
	shardLock.Lock()
	shardChannels[shardId] = recv
	shardLock.Unlock()
}
