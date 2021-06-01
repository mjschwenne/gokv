package dist_ffi

import (
	"time"
	"fmt"
	"github.com/tchajed/marshal"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"errors"
	"os"
)

type Address uint64

func (a Address) String() string {
	return AddressToStr(a)
}

func MakeAddress(ipStr string) uint64 {
	// XXX: manually parsing is pretty silly; couldn't figure out how to make
	// this work cleanly net.IP
	ipPort := strings.Split(ipStr, ":")
	if len(ipPort) != 2 {
		panic(fmt.Sprintf("Not ipv4:port %s", ipStr))
	}
	port, err := strconv.ParseUint(ipPort[1], 10, 16)
	if err != nil {
		panic(err)
	}

	ss := strings.Split(ipPort[0], ".")
	if len(ss) != 4 {
		panic(fmt.Sprintf("Not ipv4:port %s", ipStr))
	}
	ip := make([]byte, 4)
	for i, s := range ss {
		a, err := strconv.ParseInt(s, 10, 8)
		if err != nil {
			panic(err)
		}
		ip[i] = byte(a)
	}
	return (uint64(ip[0]) | uint64(ip[1])<<8 | uint64(ip[2])<<16 | uint64(ip[3])<<24 | uint64(port)<<32)
}

func AddressToStr(e Address) string {
	a0 := byte(e & 0xff)
	e = e >> 8
	a1 := byte(e & 0xff)
	e = e >> 8
	a2 := byte(e & 0xff)
	e = e >> 8
	a3 := byte(e & 0xff)
	e = e >> 8
	port := e & 0xffff
	return fmt.Sprintf("%s:%d", net.IPv4(a0, a1, a2, a3).String(), port)
}

/// Listener
type listener struct {
	l net.Listener
}

type Listener *listener

func Listen(host Address) Listener {
	l, err := net.Listen("tcp", AddressToStr(host))
	if err != nil {
		// Assume() no error on Listen. This should fail loud and early, retrying makes little sense (likely the port is already used).
		panic(err)
	}
	return &listener{l}
}

func Accept(l Listener) Connection {
	conn, err := l.l.Accept()
	if err != nil {
		// This should not usually happen... something seems wrong.
		panic(err)
	}

	return makeConnection(conn)
}

/// Connection
type connection struct {
	conn net.Conn
	send_mu *sync.Mutex // guarding *sending* on `conn`
	recv_mu *sync.Mutex // guarding *receiving* on `conn`
}

func makeConnection(conn net.Conn) Connection {
	return &connection { conn: conn, send_mu: new(sync.Mutex), recv_mu: new(sync.Mutex) }
}

type Connection *connection

type ConnectRet struct {
	Err      bool
	Connection   Connection
}

func Connect(host Address) ConnectRet {
	conn, err := net.Dial("tcp", AddressToStr(host))
	if err != nil {
		return ConnectRet { Err: true }
	}
	return ConnectRet { Err: false, Connection: makeConnection(conn) }
}

func Send(c Connection, data []byte) bool {
	// Encode length
	e := marshal.NewEnc(8)
	e.PutInt(uint64(len(data)))
	reqLen := e.Finish()

	c.send_mu.Lock()
	defer c.send_mu.Unlock()

	// message format: [dataLen] ++ data
	_, err := c.conn.Write(reqLen)
	if err == nil {
		_, err = c.conn.Write(data)
	}
	// If there was an error, make sure we never send anything on this channel again...
	// there might have been a partial write!
	if err != nil {
		c.conn.Close() // Go promises this makes this connection object "dead"
	}
	return err != nil
}

type ReceiveRet struct {
	Err    uint64 // 0 = success, 1 = timeout, 2 = other error
	Data   []byte
}

func Receive(c Connection, timeout_ms uint64) ReceiveRet {
	timeout := time.Duration(timeout_ms * 1000 * 1000) // convert to nanoseconds (Go's time unit)
	deadline := time.Now().Add(timeout)

	// Go's native locks don't support deadlines. So if there are concurrent calls
	// we might miss the above deadline. We'll still match the formal model, though,
	// and since everyone has a deadline we should get the lock *eventually*, and then
	// the pre-computed deadline above will be properly honored.
	c.recv_mu.Lock()
	defer c.recv_mu.Unlock()

	// message format: [dataLen] ++ data

	c.conn.SetReadDeadline(deadline)
	header := make([]byte, 8)
	n, err := io.ReadFull(c.conn, header)
	if err != nil {
		if n == 0 && errors.Is(err, os.ErrDeadlineExceeded) {
			// A timeout and we didn't yet read anything. So everything is in a good
			// state and we can retry later.
			return ReceiveRet { Err: 1 }
		}
		// Looks like this connection is dead.
		// This can legitimately happen when the other side "hung up", so do not panic.
		// But als, we clearly lost track here of where in the protocol we are,
		// so close it.
		c.conn.Close()
		return ReceiveRet { Err: 2 }
	}
	d := marshal.NewDec(header)
	dataLen := d.GetInt()

	data := make([]byte, dataLen)
	_, err2 := io.ReadFull(c.conn, data)
	if err2 != nil {
		// See comment above. No special handling for timeouts, as we are in the middle
		// of receiving a message so this connection is not in a good state.
		c.conn.Close()
		return ReceiveRet { Err: 2 }
	}

	return ReceiveRet { Err: 0, Data: data }
}
