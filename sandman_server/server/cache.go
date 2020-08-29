package server

import (
	"github.com/sasha-s/go-deadlock"
)

var (
	shardChannels map[string]ChannelPair
	shardLock     deadlock.RWMutex
)

func init() {
	shardChannels = make(map[string]ChannelPair)
	shardLock = deadlock.RWMutex{}
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
