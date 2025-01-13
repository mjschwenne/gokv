//--------------------------------------
// This file is autogenerated by grackle
// DO NOT MANUALLY EDIT THIS FILE
//--------------------------------------

package put_gk

import (
	"github.com/tchajed/marshal"
)

type S struct {
	OpId  uint64
	Key   string
	Value string
}

func Marshal(p S, prefix []byte) []byte {
	var enc = prefix

	enc = marshal.WriteInt(enc, p.OpId)
	keyBytes := []byte(p.Key)
	enc = marshal.WriteInt(enc, uint64(len(keyBytes)))
	enc = marshal.WriteBytes(enc, keyBytes)
	valueBytes := []byte(p.Value)
	enc = marshal.WriteInt(enc, uint64(len(valueBytes)))
	enc = marshal.WriteBytes(enc, valueBytes)

	return enc
}

func Unmarshal(s []byte) (S, []byte) {
	var enc = s // Needed for goose compatibility
	var opId uint64
	var key string
	var value string

	opId, enc = marshal.ReadInt(enc)
	var keyLen uint64
	var keyBytes []byte
	keyLen, enc = marshal.ReadInt(enc)
	keyBytes, enc = marshal.ReadBytesCopy(enc, keyLen)
	key = string(keyBytes)
	var valueLen uint64
	var valueBytes []byte
	valueLen, enc = marshal.ReadInt(enc)
	valueBytes, enc = marshal.ReadBytesCopy(enc, valueLen)
	value = string(valueBytes)

	return S{
		OpId:  opId,
		Key:   key,
		Value: value,
	}, enc
}
