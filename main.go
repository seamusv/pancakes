package main

import "flag"

var (
	clientMode = flag.Bool("c", false, "run in client mode")
)

func main() {
	flag.Parse()
	if *clientMode {
		Client()
	} else {
		Server()
	}
}
