package server

import (
	"sync"
)

var (
	shardChannels map[string]ChannelPair
	shardLock     sync.RWMutex
)

func init() {
	shardChannels = make(map[string]ChannelPair)
	shardLock = sync.RWMutex{}
}

func acquireShard(shardId string) ChannelPair {
	shardLock.RLock()
	temp := shardChannels[shardId]
	shardLock.RUnlock()
	return temp
}

func setShard(shardId string, pair ChannelPair) {
	shardLock.Lock()
	shardChannels[shardId] = pair
	shardLock.Unlock()
}

type ChannelPair struct {
	In  chan OutGoingRequest
	Out chan ASGIResponse
}
