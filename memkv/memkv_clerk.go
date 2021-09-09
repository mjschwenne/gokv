package memkv

import (
	"github.com/mit-pdos/gokv/urpc/rpc"
	"sync"
)

type MemKVCoordClerk struct {
	cl *rpc.RPCClient
}

func (ck *MemKVCoordClerk) AddShardServer(dst HostName) {
	rawRep := new([]byte)
	// TODO: on ErrDisconnect, re-create RPCClient
	for ck.cl.Call(COORD_ADD, EncodeUint64(dst), rawRep, 10000 /*ms*/) != 0 {
	}
	return
}

func (ck *MemKVCoordClerk) GetShardMap() []HostName {
	rawRep := new([]byte)
	// TODO: on ErrDisconnect, re-create RPCClient
	for ck.cl.Call(COORD_GET, make([]byte, 0), rawRep, 100 /*ms*/) != 0 {
	}
	return decodeShardMap(*rawRep)
}

// NOTE: a single clerk keeps quite a bit of state, via the shardMap[], so it
// might be good to not need to duplicate shardMap[] for a pool of clerks that's
// safe for concurrent use
type MemKVClerk struct {
	shardClerks *ShardClerkSet
	coordCk     *MemKVCoordClerk
	shardMap    []HostName // size == NSHARD; maps from sid -> host that currently owns it
}

func (ck *MemKVClerk) Get(key uint64) []byte {
	val := new([]byte)
	for {
		sid := shardOf(key)
		shardServer := ck.shardMap[sid]

		shardCk := ck.shardClerks.GetClerk(shardServer)
		err := shardCk.Get(key, val)
		if err == ENone {
			break
		}
		ck.shardMap = ck.coordCk.GetShardMap()
		continue
	}
	return *val
}

func (ck *MemKVClerk) Put(key uint64, value []byte) {
	for {
		sid := shardOf(key)
		shardServer := ck.shardMap[sid]

		shardCk := ck.shardClerks.GetClerk(shardServer)
		err := shardCk.Put(key, value)

		if err == ENone {
			break
		}
		ck.shardMap = ck.coordCk.GetShardMap()
		continue
	}
	return
}

func (ck *MemKVClerk) ConditionalPut(key uint64, expectedValue []byte, newValue []byte) bool {
	success := new(bool)
	for {
		sid := shardOf(key)
		shardServer := ck.shardMap[sid]

		shardCk := ck.shardClerks.GetClerk(shardServer)
		err := shardCk.ConditionalPut(key, expectedValue, newValue, success)

		if err == ENone {
			break
		}
		ck.shardMap = ck.coordCk.GetShardMap()
		continue
	}
	return *success
}

func (ck *MemKVClerk) Add(host HostName) {
	ck.coordCk.AddShardServer(host)
}

func multipar(num uint64, op func(uint64)) {
	var num_left = num
	num_left_mu := new(sync.Mutex)
	num_left_cond := sync.NewCond(num_left_mu)

	for i := uint64(0); i < num; i++ {
		go func() {
			op(i)
			// Signal that this one is done
			num_left_mu.Lock()
			num_left -= 1
			num_left_cond.Signal()
			num_left_mu.Unlock()
		}()
	}

	num_left_mu.Lock()
	for num_left > 0 {
		num_left_cond.Wait()
	}
	num_left_mu.Unlock()
}

// returns a slice of "values" (which are byte slices) in the same order as the
// keys passed in as input
// FIXME: benchmark
func (ck *MemKVClerk) MGet(keys []uint64) [][]byte {
	vals := make([][]byte, len(keys))
	multipar(uint64(len(keys)), func(i uint64) {
		vals[i] = ck.Get(keys[i])
	})
	return vals
}

func MakeMemKVClerk(coord HostName) *MemKVClerk {
	cck := new(MemKVCoordClerk)
	ck := new(MemKVClerk)
	ck.coordCk = cck
	ck.coordCk.cl = rpc.MakeRPCClient(coord)
	ck.shardClerks = MakeShardClerkSet()
	ck.shardMap = ck.coordCk.GetShardMap()
	return ck
}

// TODO: add an Append(key, value) (oldValue []byte) call
