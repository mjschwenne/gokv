package kv

// Replicated KV server using simplelog for durability.
// This does not use a reply table for deduplication.

import (
	"github.com/mit-pdos/gokv/grove_ffi"
	"github.com/mit-pdos/gokv/map_string_marshal"
	"github.com/mit-pdos/gokv/simplepb/simplelog"
	"github.com/tchajed/marshal"
)

type KVState struct {
	kvs map[string][]byte
}

// Ops include:
// Put(k, v)
// Get(k)
// // ConditionalPut(k, v, expected_v)
const (
	OP_PUT = byte(0)
	OP_GET = byte(1)
	// OP_CONDITIONALPUT = byte(2)
)

// begin arg structs and marshalling
type PutArgs struct {
	Key []byte
	Val []byte
}

func EncodePutArgs(args *PutArgs) []byte {
	var enc = make([]byte, 1, 1+8+uint64(len(args.Key))+uint64(len(args.Val)))
	enc[0] = OP_PUT
	enc = marshal.WriteInt(enc, uint64(len(args.Key)))
	enc = marshal.WriteBytes(enc, args.Key)
	enc = marshal.WriteBytes(enc, args.Val)
	return enc
}

func DecodePutArgs(raw_args []byte) *PutArgs {
	var enc = raw_args
	args := new(PutArgs)

	var l uint64
	l, enc = marshal.ReadInt(enc)
	args.Key = enc[:l]
	args.Val = enc[l:]

	return args
}

type getArgs []byte

func EncodeGetArgs(args getArgs) []byte {
	var enc = make([]byte, 1, 1+uint64(len(args)))
	enc[0] = OP_GET
	enc = marshal.WriteBytes(enc, args)
	return enc
}

func decodeGetArgs(raw_args []byte) getArgs {
	return raw_args
}

// end of marshalling

func (s *KVState) put(args *PutArgs) []byte {
	s.kvs[string(args.Key)] = args.Val
	return make([]byte, 0)
}

func (s *KVState) get(args getArgs) []byte {
	return s.kvs[string(args)]
}

func (s *KVState) apply(args []byte) []byte {
	if args[0] == OP_PUT {
		return s.put(DecodePutArgs(args[1:]))
	} else if args[0] == OP_GET {
		return s.get(decodeGetArgs(args[1:]))
	}
	panic("unexpected op type")
}

func (s *KVState) getState() []byte {
	return map_string_marshal.EncodeMapStringToBytes(s.kvs)
}

func (s *KVState) setState(snap []byte) {
	if len(snap) == 0 {
		s.kvs = make(map[string][]byte, 0)
	} else {
		s.kvs = map_string_marshal.DecodeMapStringToBytes(snap)
	}
}

func MakeKVStateMachine() *simplelog.InMemoryStateMachine {
	s := new(KVState)
	s.kvs = make(map[string][]byte, 0)

	return &simplelog.InMemoryStateMachine{
		ApplyVolatile: s.apply,
		GetState:      s.getState,
		SetState:      s.setState,
	}
}

func Start(fname string, me grove_ffi.Address, confHost grove_ffi.Address) {
	r := simplelog.MakePbServer(MakeKVStateMachine(), fname, confHost)
	r.Serve(me)
}
