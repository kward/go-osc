package main

import (
	"fmt"

	"github.com/kward/go-osc/osc"
)

func main() {
	addr := "127.0.0.1:8000"
	server, err := osc.NewServer(addr)
	if err != nil {
		panic(err)
	}

	server.Handle("/message/address", func(msg *osc.Message) {
		fmt.Println(msg)
	})

	server.ListenAndServe()
}
