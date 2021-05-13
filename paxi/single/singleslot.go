package single

import (
	"sync"
)

type Entry = uint64

type Replica struct {
	mu         *sync.Mutex
	promisedPN uint64 // server has promised not to accept proposals below this

	logPN uint64  // proposal number of accepted val
	log   []Entry // the value itself

	isLeader bool // this means that we own the proposal with number logPN

	commitIndex   uint64
	acceptedIndex uint64

	peers []*Clerk
}

type PrepareReply struct {
	Success bool
	Log     []Entry // full log;
	Pn      uint64
}

func (r *Replica) PrepareRPC(pn uint64, reply *PrepareReply) {
	r.mu.Lock()
	if pn > r.promisedPN {
		r.promisedPN = pn

		reply.Pn = r.logPN
		reply.Log = r.log
		reply.Success = true
	} else {
		reply.Success = false
		reply.Pn = r.promisedPN
	}
	r.mu.Unlock()
}

type ProposeArgs struct {
	Pn          uint64
	CommitIndex uint64
	Log         []Entry
}

func (r *Replica) ProposeRPC(pn uint64, commitIndex uint64, val []Entry) bool {
	r.mu.Lock()
	if pn >= r.promisedPN && pn >= r.logPN {
		if pn > r.logPN {
			r.log = val
			r.logPN = pn
		} else if len(val) > len(r.log) {
			r.log = val
		}
		if commitIndex > r.commitIndex {
			r.commitIndex = commitIndex
		}
		r.mu.Unlock()
		return true
	} else {
		r.mu.Unlock()
		return false
	}
}

func (r *Replica) Start(cmd Entry) {
}

func (r *Replica) GetLog() []Entry {
	return nil
}

// returns true iff there was an error
func (r *Replica) TryDecide(v ValType, outv *ValType) bool {
	r.mu.Lock()
	pn := r.promisedPN + 1 // don't need to bother incrementing; will invoke RPC on ourselves
	r.mu.Unlock()

	var numPrepared uint64
	numPrepared = 0
	var highestPn uint64
	highestPn = 0
	var highestVal ValType
	highestVal = v // if no one in our majority has accepted a value, we'll propose this one
	mu := new(sync.Mutex)

	for _, peer := range r.peers { // XXX: peers is readonly
		local_peer := peer
		go func() {
			reply_ptr := new(PrepareReply)
			local_peer.Prepare(pn, reply_ptr) // TODO: replace with real RPC

			if reply_ptr.Success {
				mu.Lock()
				numPrepared = numPrepared + 1
				if reply_ptr.Pn > highestPn {
					highestVal = reply_ptr.Val
					highestPn = reply_ptr.Pn
				}
				mu.Unlock()
			}
		}()
	}

	// FIXME: put this in a condvar loop with timeout
	mu.Lock()
	n := numPrepared
	proposeVal := highestVal
	mu.Unlock()

	if 2*n > uint64(len(r.peers)) {
		mu2 := new(sync.Mutex)
		var numAccepted uint64
		numAccepted = 0

		for _, peer := range r.peers {
			local_peer := peer
			// each thread talks to a unique peer
			go func() {
				r := local_peer.Propose(pn, proposeVal) // TODO: replace with real RPC
				if r {
					mu2.Lock()
					numAccepted = numAccepted + 1
					mu2.Unlock()
				}
			}()
		}

		mu2.Lock()
		n := numAccepted
		mu2.Unlock()

		if 2*n > uint64(len(r.peers)) {
			*outv = proposeVal
			return false
		} else {
			return true
		}
	} else {
		return true
	}
}