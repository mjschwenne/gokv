package asyncfile

import (
	"sync"

	"github.com/mit-pdos/gokv/grove_ffi"
)

type File struct {
	mu               *sync.Mutex
	data             []byte
	filename         string
	index            uint64
	indexCond        *sync.Cond
	durableIndex     uint64
	durableIndexCond *sync.Cond

	closeRequested bool
	closed         bool
	closedCond     *sync.Cond
}

func (s *File) AsyncWrite(data []byte) func() {
	// XXX: can read index here because it's owned by the owner of the File object
	s.mu.Lock()
	s.data = data
	s.index += 1
	index := s.index
	s.mu.Unlock()
	return func() { s.wait(index) }
}

func (s *File) wait(index uint64) {
	s.mu.Lock()
	for s.durableIndex < index {
		s.durableIndexCond.Wait()
	}
	s.mu.Unlock()
}

func (s *File) flushThread() {
	s.mu.Lock()
	for {
		if s.closeRequested {
			// flush everything and exit
			grove_ffi.FileWrite(s.filename, s.data)
			s.durableIndex = s.index
			s.durableIndexCond.Broadcast()
			s.closed = true
			s.mu.Unlock()
			s.closedCond.Signal()
			break
		}
		if s.durableIndex >= s.index {
			s.indexCond.Wait()
			continue
		}
		index := s.index
		data := s.data
		s.mu.Unlock()
		grove_ffi.FileWrite(s.filename, data)
		s.mu.Lock()
		s.durableIndex = index
		s.durableIndexCond.Broadcast()
		// TODO: can avoid false wakeups by having two condvars like aof
	}
}

func (s *File) Close() {
	s.mu.Lock()
	s.closeRequested = true
	s.indexCond.Signal()
	for !s.closed {
		s.closedCond.Wait()
	}
	s.mu.Unlock()
}

// returns the state, then the File object
func MakeFile(filename string) ([]byte, *File) {
	s := new(File)
	s.mu = new(sync.Mutex)
	s.indexCond = sync.NewCond(s.mu)
	s.durableIndexCond = sync.NewCond(s.mu)
	s.closedCond = sync.NewCond(s.mu)
	s.filename = filename

	s.data = grove_ffi.FileRead(filename)
	s.index = 0
	s.durableIndex = 0
	s.closed = false
	s.closeRequested = false

	go func() { s.flushThread() }()

	return s.data, s
}
