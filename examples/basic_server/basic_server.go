package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/kward/go-osc/osc"
)

func main() {
	addr := "127.0.0.1:8000"
	server, err := osc.NewServer(addr)
	if err != nil {
		fmt.Println("Error creating server:", err)
		os.Exit(1)
	}

	fmt.Println("### Welcome to go-osc receiver demo")
	fmt.Println("Press \"q\" to exit")

	// Add a catch-all handler that prints all incoming messages
	err = server.Handle("/", func(msg *osc.Message) {
		fmt.Println("-- OSC Message:", msg)
	})
	if err != nil {
		fmt.Println("Error adding handler:", err)
		os.Exit(1)
	}

	// Start the server in a goroutine
	go func() {
		fmt.Println("Start listening on", addr)
		if err := server.ListenAndServe(); err != nil {
			fmt.Println("Server error:", err)
			os.Exit(1)
		}
	}()

	reader := bufio.NewReader(os.Stdin)

	for {
		c, err := reader.ReadByte()
		if err != nil {
			os.Exit(0)
		}

		if c == 'q' {
			os.Exit(0)
		}
	}
}
