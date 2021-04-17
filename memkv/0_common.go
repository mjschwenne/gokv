package memkv

import (
	"github.com/tchajed/marshal"
	"github.com/mit-pdos/lockservice/grove_ffi"
)

type HostName = grove_ffi.HostName

type ValueType = uint64

type ErrorType = uint64

const (
	ENone = uint64(0)
	EDontHaveShard = uint64(1)
)

const NSHARD = uint64(65536)

// rpc ids
const KV_FRESHCID = uint64(0)
const KV_PUT = uint64(1)
const KV_GET = uint64(2)
const KV_INS_SHARD = uint64(3)
const KV_MOV_SHARD = uint64(4)

func shardOf(key uint64) uint64 {
	return key % NSHARD
}

type PutRequest struct {
	CID   uint64
	Seq   uint64
	Key   uint64
	Value []byte
}

// doesn't include the operation type
func encodePutRequest(args *PutRequest) []byte {
	num_bytes := uint64(8 + 8 + 8 + 8 + len(args.Value)) // CID + Seq + key + value-len + value
	// num_bytes = uint64(8 + len(args.Value))
	e := marshal.NewEnc(num_bytes)
	e.PutInt(args.CID)
	e.PutInt(args.Seq)
	e.PutInt(args.Key)
	e.PutInt(uint64(len(args.Value)))
	e.PutBytes(args.Value)

	return e.Finish()
}

func decodePutRequest(reqData []byte) *PutRequest {
	req := new(PutRequest)
	d := marshal.NewDec(reqData)
	req.CID = d.GetInt()
	req.Seq = d.GetInt()
	req.Key = d.GetInt()
	req.Value = d.GetBytes(d.GetInt())

	return req
}

type PutReply struct {
	Err ErrorType
}

func encodePutReply(reply *PutReply) []byte {
	e := marshal.NewEnc(8)
	e.PutInt(reply.Err)
	return e.Finish()
}

func decodePutReply(replyData []byte) *PutReply {
	reply := new(PutReply)
	d := marshal.NewDec(replyData)
	reply.Err = d.GetInt()
	return reply
}

type GetRequest struct {
	CID uint64
	Seq uint64
	Key uint64
}

type GetReply struct {
	Err   ErrorType
	Value []byte
}

func encodeGetRequest(req *GetRequest) []byte {
	e := marshal.NewEnc(3 * 8)
	e.PutInt(req.CID)
	e.PutInt(req.Seq)
	e.PutInt(req.Key)
	return e.Finish()
}

func decodeGetRequest(rawReq []byte) *GetRequest {
	req := new(GetRequest)
	d := marshal.NewDec(rawReq)
	req.CID = d.GetInt()
	req.Seq = d.GetInt()
	req.Key = d.GetInt()
	return req
}

func encodeGetReply(rep *GetReply) []byte {
	num_bytes := uint64(8 + 8 + len(rep.Value)) // CID + Seq + key + value-len + value
	e := marshal.NewEnc(num_bytes)
	e.PutInt(rep.Err)
	e.PutInt(uint64(len(rep.Value)))
	e.PutBytes(rep.Value)
	return e.Finish()
}

func decodeGetReply(rawRep []byte) *GetReply {
	rep := new(GetReply)
	d := marshal.NewDec(rawRep)
	rep.Err = d.GetInt()
	rep.Value = d.GetBytes(d.GetInt())

	return rep
}

type InstallShardRequest struct {
	CID uint64
	Seq uint64
	Sid uint64
	Kvs map[uint64][]byte
}

func encodeInstallShardRequest(req *InstallShardRequest) []byte {
	// TODO: deal with map encoding; maybe put try to get it in tchajed/marshal?
	panic("unimpl")
}

func decodeInstallShardRequest(rawReq []byte) *InstallShardRequest {
	// TODO: deal with map encoding; maybe put try to get it in tchajed/marshal?
	panic("unimpl")
}

type MoveShardRequest struct {
	Sid uint64
	Dst HostName
}

func encodeMoveShardRequest(req *MoveShardRequest) []byte {
	panic("unimpl")
}

func decodeMoveShardRequest(rawReq []byte) *MoveShardRequest {
	panic("unimpl")
}

func encodeCID(cid uint64) []byte {
	e := marshal.NewEnc(8)
	e.PutInt(cid)
	return e.Finish()
}

func decodeCID(rawCID []byte) uint64 {
	return marshal.NewDec(rawCID).GetInt()
}

func encodeShardMap(shardMap *[]HostName) []byte {
	panic("unimpl")
}

func decodeShardMap(raw []byte) []HostName {
	panic("unimpl")
}