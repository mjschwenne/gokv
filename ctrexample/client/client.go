package main

import (
	"fmt"
	"github.com/mit-pdos/gokv/urpc/rpc"
	"github.com/tchajed/goose/machine"
	"github.com/tchajed/marshal"
)

const (
	FAI_OP = uint64(0)
)

// the boot/main() function for the server
func main() {
	cl := rpc.MakeRPCClient(53021371269120) // hardcoded "127.0.0.1:12345"

	// FIXME: client needs to try reconnecting; could use connman to make that so.
	var localBound = uint64(0)
	for {
		rep := new([]byte)
		err := cl.Call(FAI_OP, make([]byte, 0), rep, 100 /* ms */)
		if err != 0 {
			continue // failed, just retry
		}
		dec := marshal.NewDec(*rep)
		v := dec.GetInt()

		machine.Assert(v >= localBound)
		localBound = v
		fmt.Println("One")
	}
}
