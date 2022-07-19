package main

import (
	"flag"
	"fmt"
	"github.com/mit-pdos/gokv/grove_ffi"
	"github.com/mit-pdos/gokv/simplepb/admin"
	"os"
)

func main() {
	var confStr string
	flag.StringVar(&confStr, "conf", "", "address of config server")
	flag.Parse()

	usage_assert := func(b bool) {
		if !b {
			flag.PrintDefaults()
			fmt.Println("Must provide command in form:")
			fmt.Println(" init host1 [host2 ...]")
			fmt.Println(" reconfig host1 [host2 ...]")
			os.Exit(1)
		}
	}

	usage_assert(confStr != "")

	confHost := grove_ffi.MakeAddress(confStr)

	a := flag.Args()
	usage_assert(len(a) > 0)
	if a[0] == "init" {
		servers := make([]grove_ffi.Address, 0)
		for _, srvStr := range a[1:] {
			servers = append(servers, grove_ffi.MakeAddress(srvStr))
		}
		admin.InitializeSystem(confHost, servers)
		fmt.Printf("Initialized system")
	} else if a[0] == "reconfig" {
		servers := make([]grove_ffi.Address, 0)
		for _, srvStr := range a[1:] {
			servers = append(servers, grove_ffi.MakeAddress(srvStr))
		}
		admin.EnterNewConfig(confHost, servers)
		fmt.Printf("Finished switching configuration")
	}
}
