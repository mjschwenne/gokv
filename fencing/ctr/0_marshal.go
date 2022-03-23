package ctr

import (
	"github.com/tchajed/marshal"
)

type PutArgs struct {
	cid   uint64
	seq   uint64
	epoch uint64
	v     uint64
}

func EncPutArgs(args *PutArgs) []byte {
	enc := marshal.NewEnc(uint64(24))
	enc.PutInt(args.cid)
	enc.PutInt(args.seq)
	enc.PutInt(args.v)
	enc.PutInt(args.epoch)
	return enc.Finish()
}

func DecPutArgs(raw_args []byte) *PutArgs {
	dec := marshal.NewDec(raw_args)
	args := new(PutArgs)
	args.cid = dec.GetInt()
	args.seq = dec.GetInt()
	args.v = dec.GetInt()
	args.epoch = dec.GetInt()
	return args
}

type GetArgs struct {
	cid   uint64
	seq   uint64
	epoch uint64
}

func EncGetArgs(args *GetArgs) []byte {
	enc := marshal.NewEnc(uint64(24))
	enc.PutInt(args.cid)
	enc.PutInt(args.seq)
	enc.PutInt(args.epoch)
	return enc.Finish()
}

func DecGetArgs(raw_args []byte) *GetArgs {
	dec := marshal.NewDec(raw_args)
	args := new(GetArgs)
	args.cid = dec.GetInt()
	args.seq = dec.GetInt()
	args.epoch = dec.GetInt()
	return args
}

type GetReply struct {
	epoch uint64
	val   uint64
}

func EncGetReply(args *GetReply) []byte {
	enc := marshal.NewEnc(uint64(16))
	enc.PutInt(args.epoch)
	enc.PutInt(args.val)
	return enc.Finish()
}

func DecGetReply(raw_args []byte) *GetReply {
	dec := marshal.NewDec(raw_args)
	args := new(GetReply)
	args.epoch = dec.GetInt()
	args.val = dec.GetInt()
	return args
}