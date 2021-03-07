package gokv

import (
	"github.com/tchajed/marshal"
	"io/ioutil"
	"sync"
)

type GoKVServer struct {
	mu        *sync.Mutex
	lastReply map[uint64]uint64
	lastSeq   map[uint64]uint64
	kvs       map[uint64]string
	kvsSize   uint64
	durable   bool
	opLog     *AppendableFile
}

type PutArgs struct {
	Key   uint64
	Value string
}

// XXX: for correctness under packet duplication, should have generation number
func (s *GoKVServer) ResetRPC(args *struct{}, reply *struct{}) error {
	s.mu.Lock()

	s.lastReply = make(map[uint64]uint64)
	s.lastSeq = make(map[uint64]uint64)
	s.kvs = make(map[uint64]string)
	s.kvsSize = 8
	WriteDurableKVServer(s)

	s.mu.Unlock()
	return nil
}

func (s *GoKVServer) appendPut(args *PutArgs) {
	v := []byte(args.Value)
	num_bytes := uint64(8 + 8 + len(v)) // key + value-len + value
	e := marshal.NewEnc(num_bytes)
	e.PutInt(args.Key)
	e.PutInt(uint64(len(v)))
	e.PutBytes(v)
	s.opLog.Append(e.Finish())
}

func (s *GoKVServer) PutRPC(args *PutArgs, reply *struct{}) error {
	s.mu.Lock()
	oldv, ok := s.kvs[args.Key]
	if ok {
		s.kvsSize -= uint64(len([]byte(oldv)))
		s.kvsSize += uint64(len([]byte(args.Value)))
	} else {
		s.kvsSize += uint64(len([]byte(args.Value)))
		s.kvsSize += 8
		s.kvsSize += 8
	}
	s.kvs[args.Key] = args.Value
	if s.durable {
		s.mu.Unlock()
		s.appendPut(args)
		return nil
	}
	s.mu.Unlock()
	return nil
}

func (s *GoKVServer) GetRPC(key *uint64, value *string) error {
	s.mu.Lock()
	*value = s.kvs[*key]
	s.mu.Unlock()
	return nil
}

func EncMap(e *marshal.Enc, m map[uint64]uint64) {
	e.PutInt(uint64(len(m)))
	for key, value := range m {
		e.PutInt(key)
		e.PutInt(value)
	}
}

// requires ?some? amount of space
func EncByteMap(e *marshal.Enc, m map[uint64]string) {
	e.PutInt(uint64(len(m)))
	for key, value := range m {
		e.PutInt(key)
		e.PutInt(uint64(len(value)))
		e.PutBytes([]byte(value))
	}
}

func WriteDurableKVServer(ks *GoKVServer) {
	num_bytes := uint64(8*(2*len(ks.lastSeq)+2*len(ks.lastReply)+2)) + ks.kvsSize
	e := marshal.NewEnc(num_bytes) // 4 uint64s
	EncMap(&e, ks.lastSeq)
	EncMap(&e, ks.lastReply)
	EncByteMap(&e, ks.kvs)

	// TODO: this isn't crash-atomic
	ioutil.WriteFile("kvdur", e.Finish(), 0644)
	return
}

func MakeGoKVServer() *GoKVServer {
	srv := new(GoKVServer)
	srv.mu = new(sync.Mutex)
	srv.lastReply = make(map[uint64]uint64)
	srv.lastSeq = make(map[uint64]uint64)
	srv.kvs = make(map[uint64]string)
	srv.kvsSize = 8 // for len
	srv.durable = true
	srv.opLog = CreateAppendableFile("kvdur_log")
	return srv
}
