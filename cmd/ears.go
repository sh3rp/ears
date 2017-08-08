package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/sh3rp/ears"
)

var format string
var timeoutMS int64
var intf string

func main() {

	if len(os.Args) < 2 {
		log.Fatal().Msgf("You must supply an interface name.")
		os.Exit(1)
	}

	flag.StringVar(&format, "f", "list", "Output format [list|json|jsonall]")
	flag.Int64Var(&timeoutMS, "t", 3000, "Timeout in milliseconds")
	flag.StringVar(&intf, "i", "", "Interface name to use for sending pings")
	flag.Parse()

	if intf == "" {
		fmt.Printf("You must specify an interface (use -i, example: -i \"eth0\")\n")
		os.Exit(1)
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
