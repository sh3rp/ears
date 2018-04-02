package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/sh3rp/ears"
)

var format string
var timeoutMS int64
var intf string

var ignore_intfs []string = []string{"br-", "docker0"}

func main() {

	flag.StringVar(&format, "f", "list", "Output format [list|json|jsonall]")
	flag.Int64Var(&timeoutMS, "t", 3000, "Timeout in milliseconds")
	flag.StringVar(&intf, "i", "", "Interface name to use for sending pings")
	flag.Parse()

	if intf == "" {
		foundIntf := false
		intfs, _ := net.Interfaces()
		for _, i := range intfs {
			var ignoreIntf bool
			// ignore down or loopback
			if (i.Flags&net.FlagUp) == 0 || (i.Flags&net.FlagLoopback) != 0 {
				continue
			}
			// ignore docker bridges
			for _, ignore := range ignore_intfs {
				if strings.HasPrefix(i.Name, ignore) {
					ignoreIntf = true
					break
				}
			}
			if ignoreIntf {
				continue
			}
			addrs, _ := i.Addrs()
			for _, a := range addrs {
				if strings.Contains(a.String(), ".") {
					intf = i.Name
					foundIntf = true
					break
				}
			}
			if foundIntf {
				break
			}
		}
		if !foundIntf {
			fmt.Printf("Unable to automatically determine network interface, specify using -i (e.g. -i eth0)")
			os.Exit(1)
		}
	}

	pinger := ears.NewPinger(timeoutMS)

	pinger.PingIPHosts(intf)

	switch format {
	case "list":
		for _, v := range pinger.Pings {
			if v.Alive {
				fmt.Printf("%s\n", v.IP)
			}
		}
	case "jsonall":
		json, err := json.Marshal(pinger.Pings)
		if err != nil {
			log.Error().Msgf("Error marshalling JSON: %v", err)
		}
		fmt.Println(string(json))
	case "json":
		pings := pinger.Pings
		alivePings := make(map[string]*ears.Ping)
		for k, v := range pings {
			if v.Alive {
				alivePings[k] = v
			}
		}
		json, err := json.Marshal(alivePings)
		if err != nil {
			log.Error().Msgf("Error marshalling JSON: %v", err)
		}
		fmt.Println(string(json))
	}
}
