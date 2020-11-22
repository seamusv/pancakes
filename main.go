package main

import (
	"flag"
	"log"
)

var (
	clientMode = flag.Bool("c", false, "run in client mode")
)

func main() {
	flag.Parse()

	var err error
	if *clientMode {
		err = Client()
	} else {
		err = Server()
	}

	if err != nil {
		log.Print(err)
	}
}
