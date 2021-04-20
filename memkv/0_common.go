package memkv

import (
	"github.com/tchajed/marshal"
)

type HostName = string

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

// NOTE: probably can just amortize this by keeping track of this with the map itself
func SizeOfMarshalledMap(m map[uint64][]byte) uint64 {
	s := uint64(8)
	for _, value := range m {
		s += (uint64(len(value)) + 8 + 8)
	}
	return s
}

func EncSliceMap(e marshal.Enc, m map[uint64][]byte) {
	e.PutInt(uint64(len(m)))
	for key, value := range m {
		e.PutInt(key)
		e.PutInt(uint64(len(value)))
		e.PutBytes(value)
	}
}

func DecSliceMap(d marshal.Dec) map[uint64][]byte {
	sz := d.GetInt()
	m := make(map[uint64][]byte)
	var i = uint64(0)
	for i < sz {
		k := d.GetInt()
		v := d.GetBytes(d.GetInt())
		m[k] = v
		i = i + 1
	}
	return m
}

func encodeInstallShardRequest(req *InstallShardRequest) []byte {
	e := marshal.NewEnc(8 + 8 + 8 + SizeOfMarshalledMap(req.Kvs) )
	e.PutInt(req.CID)
	e.PutInt(req.Seq)
	e.PutInt(req.Sid)
	EncSliceMap(e, req.Kvs)
	return e.Finish()
}

func decodeInstallShardRequest(rawReq []byte) *InstallShardRequest {
	d := marshal.NewDec(rawReq)
	req := new(InstallShardRequest)
	req.CID = d.GetInt()
	req.Seq = d.GetInt()
	req.Sid = d.GetInt()
	req.Kvs = DecSliceMap(d)
	return req
}

type MoveShardRequest struct {
	Sid uint64
	Dst HostName
}

func encodeMoveShardRequest(req *MoveShardRequest) []byte {
	e := marshal.NewEnc(8*2)
	e.PutInt(req.Sid)
	v := []byte(req.Dst)
	e.PutInt(uint64(len(v)))
	e.PutBytes(v)
	return e.Finish()
}

func decodeMoveShardRequest(rawReq []byte) *MoveShardRequest {
	req := new(MoveShardRequest)
	d := marshal.NewDec(rawReq)
	req.Sid = d.GetInt()
	req.Dst = string(d.GetBytes(d.GetInt()))
	return req
}

func encodeCID(cid uint64) []byte {
	e := marshal.NewEnc(8)
	e.PutInt(cid)
	return e.Finish()
}

func decodeCID(rawCID []byte) uint64 {
	return marshal.NewDec(rawCID).GetInt()
}

func marshalledShardMapSize(shardMap *[]HostName) uint64 {
	s := uint64(0)
	for i := uint64(0); i < NSHARD; i++ {
		s += 8
		s += uint64(len([]byte((*shardMap)[i])))
	}
	return s
}

func encodeShardMap(shardMap *[]HostName) []byte {
	// requires that shardMap is a list
	num_bytes := marshalledShardMapSize(shardMap)
	e := marshal.NewEnc(num_bytes)
	for i := uint64(0); i < NSHARD; i++ {
		v := []byte((*shardMap)[i])
		e.PutInt(uint64(len(v)))
		e.PutBytes(v)
	}
	return e.Finish()
}

func decodeShardMap(raw []byte) []HostName {
	shardMap := make([]HostName, NSHARD)
	d := marshal.NewDec(raw)
	for i := uint64(0); i < NSHARD; i++ {
		shardMap[i] = string(d.GetBytes(d.GetInt()))
	}
	return shardMap
}
