package mpaxos

import (
	"github.com/mit-pdos/gokv/grove_ffi"
)

const (
	RPC_APPLY_AS_FOLLOWER = uint64(0)
	RPC_ENTER_NEW_EPOCH   = uint64(1)
	RPC_APPLY             = uint64(2)
	RPC_BECOME_LEADER     = uint64(3)
)

// these clerks hide connection failures, and retry forever
type singleClerk struct {
	cl   *ReconnectingClient
	addr grove_ffi.Address
}

func makeSingleClerk(addr grove_ffi.Address) *singleClerk {
	// make a bunch of urpc clients
	ck := &singleClerk{
		cl: MakeReconnectingClient(addr),
	}

	return ck
}

func (s *singleClerk) enterNewEpoch(args *enterNewEpochArgs) *enterNewEpochReply {
	raw_args := encodeEnterNewEpochArgs(args)
	raw_reply := new([]byte)
	err := s.cl.Call(RPC_ENTER_NEW_EPOCH, raw_args, raw_reply, 500 /* ms */)
	if err == 0 {
		return decodeEnterNewEpochReply(*raw_reply)
	} else {
		return &enterNewEpochReply{err: ETimeout}
	}

}

func (s *singleClerk) applyAsFollower(args *applyAsFollowerArgs) *applyAsFollowerReply {
	raw_args := encodeApplyAsFollowerArgs(args)
	raw_reply := new([]byte)
	err := s.cl.Call(RPC_APPLY_AS_FOLLOWER, raw_args, raw_reply, 500 /* ms */)
	if err == 0 {
		return decodeApplyAsFollowerReply(*raw_reply)
	} else {
		return &applyAsFollowerReply{err: ETimeout}
	}
}

func (s *singleClerk) becomeLeader() {
	// make the server the primary
	reply := new([]byte)
	s.cl.Call(RPC_BECOME_LEADER, make([]byte, 0), reply, 500 /* ms */)
}

func (s *singleClerk) apply(op []byte) (Error, []byte) {
	reply := new([]byte)
	// tell the server to apply the op
	err2 := s.cl.Call(RPC_APPLY, op, reply, 500 /* ms*/)
	if err2 != 0 {
		return ETimeout, nil
	}

	r := decodeApplyReply(*reply)
	if r.err != ENone {
		return r.err, nil
	}

	return ENone, r.ret
}
