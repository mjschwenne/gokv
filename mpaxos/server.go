package mpaxos

import (
	"log"
	"sync"

	"github.com/mit-pdos/gokv/grove_ffi"
	"github.com/mit-pdos/gokv/urpc"
)

type Server struct {
	mu            *sync.Mutex
	epoch         uint64
	acceptedEpoch uint64

	nextIndex uint64
	state     []byte

	clerks   []*singleClerk
	isLeader bool

	applyFn func(state []byte, op []byte) ([]byte, []byte)
}

func (s *Server) applyAsFollower(args *applyAsFollowerArgs, reply *applyAsFollowerReply) {
	s.mu.Lock()
	if s.epoch == args.epoch {
		if s.nextIndex == args.nextIndex {
			s.nextIndex += 1
			s.state = args.state
			reply.err = ENone
		} else if s.nextIndex < args.nextIndex {
			reply.err = ENone
		}
	} else if s.epoch < args.epoch {
		s.epoch = args.epoch
		s.state = args.state
		s.nextIndex = args.nextIndex
		reply.err = ENone
	} else {
		reply.err = EEpochStale
	}
	s.mu.Unlock()
}

// FIXME:
// This will vote yes only the first time it's called in an epoch.
// If you have too aggressive of a timeout and end up retrying this, the retry
// might fail because it may be the second execution of enterNewEpoch(epoch) on
// the server.
// Solution: either conservative (maybe double) timeouts, or don't use this for
// leader election, only for coming up with a valid proposal.
func (s *Server) enterNewEpoch(args *enterNewEpochArgs, reply *enterNewEpochReply) {
	s.mu.Lock()
	if s.epoch >= args.epoch {
		s.mu.Unlock()
		reply.err = EEpochStale
		return
	}
	// else, s.epoch < args.epoch
	s.epoch = args.epoch
	reply.acceptedEpoch = s.acceptedEpoch
	reply.nextIndex = s.nextIndex
	reply.state = s.state
	s.mu.Unlock()
}

func (s *Server) becomeLeader() {
	log.Println("started trybecomeleader")
	s.mu.Lock()
	if s.isLeader {
		s.mu.Unlock()
		return
	}
	// pick a new epoch number
	s.isLeader = false
	clerks := s.clerks
	args := &enterNewEpochArgs{epoch: s.epoch + 1}
	s.mu.Unlock()

	var numReplies = uint64(0)
	replies := make([]enterNewEpochReply, uint64(len(clerks)))

	var i = uint64(0)
	n := uint64(len(replies))
	for i < n {
		replies[i].err = ETimeout
	}

	mu := new(sync.Mutex)
	numReplies_cond := sync.NewCond(mu)
	q := uint64(len(clerks)+1) / 2

	for i, ck := range clerks {
		ck := ck
		i := i
		go func() {
			reply := new(enterNewEpochReply)
			ck.enterNewEpoch(args, reply)
			mu.Lock()
			numReplies += 1
			replies[i] = *reply
			if numReplies >= q {
				numReplies_cond.Signal()
			}
			mu.Unlock()
		}()
	}

	mu.Lock()
	// wait for a quorum of replies
	for numReplies < q {
		numReplies_cond.Wait()
	}

	var latestReply enterNewEpochReply
	var numSuccesses = uint64(0)
	for _, reply := range replies {
		if reply.err == ENone {
			numSuccesses += 1
			if latestReply.acceptedEpoch < reply.acceptedEpoch {
				latestReply = reply
			} else if latestReply.acceptedEpoch == reply.acceptedEpoch &&
				reply.nextIndex > latestReply.nextIndex {
				latestReply = reply
			}
		}
	}

	log.Println("finished trybecomeleader")
	if numSuccesses >= q {
		log.Println("succeeded becomeleader")
		s.mu.Lock() // RULE: lock s.mu after mu
		if s.epoch < args.epoch {
			s.isLeader = true
			s.acceptedEpoch = s.epoch
			s.state = latestReply.state
		}
		s.mu.Unlock()
		mu.Unlock()
	} else {
		mu.Unlock()
		log.Println("failed becomeleader")
		// failed
	}
}

func (s *Server) apply(op []byte, reply *applyReply) {
	s.mu.Lock()
	if !s.isLeader {
		s.mu.Unlock()
		reply.err = ENotLeader
		return
	}
	s.state, reply.ret = s.applyFn(s.state, op)
	args := &applyAsFollowerArgs{epoch: s.epoch, nextIndex: s.nextIndex, state: s.state}
	s.nextIndex += 1
	clerks := s.clerks
	s.mu.Unlock()

	var numReplies = uint64(0)
	replies := make([]*applyAsFollowerReply, uint64(len(clerks)))
	mu := new(sync.Mutex)
	numReplies_cond := sync.NewCond(mu)
	q := uint64((len(clerks)+1)+1) / 2

	for i, ck := range clerks {
		ck := ck
		i := i
		go func() {
			reply := new(applyAsFollowerReply)
			ck.applyAsFollower(args, reply)

			mu.Lock()
			numReplies += 1
			replies[i] = reply
			if numReplies >= q {
				numReplies_cond.Signal()
			}
			mu.Unlock()
		}()
	}

	mu.Lock()
	// wait for a quorum of replies
	for numReplies < q {
		numReplies_cond.Wait()
	}

	var numSuccesses = uint64(0)
	for _, reply := range replies {
		if reply.err == ENone {
			numSuccesses += 1
		}
	}

	if numSuccesses >= q {
		reply.err = ENone
	} else {
		reply.err = EEpochStale
	}
}

func makeServer(fname string, applyFn func([]byte, []byte) ([]byte, []byte),
	config []grove_ffi.Address) *Server {
	s := new(Server)
	s.mu = new(sync.Mutex)

	s.state = make([]byte, 0)

	s.clerks = make([]*singleClerk, len(config))
	n := uint64(len(s.clerks))
	var i = uint64(0)
	for i < n {
		s.clerks[i] = makeSingleClerk(config[i])
		i += 1
	}
	return s
}

func StartServer(fname string, me grove_ffi.Address,
	applyFn func([]byte, []byte) ([]byte, []byte),
	config []grove_ffi.Address) {
	s := makeServer(fname, applyFn, config)

	handlers := make(map[uint64]func([]byte, *[]byte))
	handlers[RPC_APPLY_AS_FOLLOWER] = func(raw_args []byte, raw_reply *[]byte) {
		reply := new(applyAsFollowerReply)
		args := decodeApplyAsFollowerArgs(raw_args)
		s.applyAsFollower(args, reply)
		*raw_reply = encodeApplyAsFollowerReply(reply)
	}

	handlers[RPC_ENTER_NEW_EPOCH] = func(raw_args []byte, raw_reply *[]byte) {
		reply := new(enterNewEpochReply)
		args := decodeEnterNewEpochArgs(raw_args)
		s.enterNewEpoch(args, reply)
		*raw_reply = encodeEnterNewEpochReply(reply)
	}

	handlers[RPC_APPLY] = func(raw_args []byte, raw_reply *[]byte) {
		reply := new(applyReply)
		s.apply(raw_args, reply)
		*raw_reply = encodeApplyReply(reply)
	}

	handlers[RPC_BECOME_LEADER] = func(raw_args []byte, raw_reply *[]byte) {
		s.becomeLeader()
	}

	r := urpc.MakeServer(handlers)
	r.Serve(me)
}
