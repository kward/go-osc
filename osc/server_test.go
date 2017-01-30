package osc

import (
	"net"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/context"
)

func TestHandle(t *testing.T) {
	server, err := NewServer("localhost:6677")
	if err != nil {
		t.Errorf("unexpected error; %s", err)
	}
	if err := server.Handle("/address/test", func(msg *Message) {}); err != nil {
		t.Error("Expected that OSC address '/address/test' is valid")
	}
}

func TestHandleWithInvalidAddress(t *testing.T) {
	server, err := NewServer("localhost:6677")
	if err != nil {
		t.Errorf("unexpected error; %s", err)
	}
	if err := server.Handle("/address*/test", func(msg *Message) {}); err == nil {
		t.Error("Expected error with '/address*/test'")
	}
}

func TestMessageDispatching(t *testing.T) {
	finish := make(chan bool)
	start := make(chan bool)
	done := sync.WaitGroup{}
	done.Add(2)

	// Start the OSC server in a new go-routine
	go func() {
		conn, err := net.ListenPacket("udp", "localhost:6677")
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()

		server, err := NewServer("localhost:6677")
		if err != nil {
			t.Fatal(err)
		}
		err = server.Handle("/address/test", func(msg *Message) {
			if len(msg.Arguments) != 1 {
				t.Error("Argument length should be 1 and is: " + string(len(msg.Arguments)))
			}

			if msg.Arguments[0].(int32) != 1122 {
				t.Error("Argument should be 1122 and is: " + string(msg.Arguments[0].(int32)))
			}

			// Stop OSC server
			conn.Close()
			finish <- true
		})
		if err != nil {
			t.Error("Error adding message handler")
		}

		start <- true
		server.Serve(context.Background(), conn)
	}()

	go func() {
		timeout := time.After(5 * time.Second)
		select {
		case <-timeout:
		case <-start:
			time.Sleep(500 * time.Millisecond)
			client := NewClient("localhost", 6677)
			msg := NewMessage("/address/test")
			msg.Append(int32(1122))
			client.Send(msg)
		}

		done.Done()

		select {
		case <-timeout:
		case <-finish:
		}
		done.Done()
	}()

	done.Wait()
}

func TestMessageReceiving(t *testing.T) {
	finish := make(chan bool)
	start := make(chan bool)
	done := sync.WaitGroup{}
	done.Add(2)

	// Start the server in a go-routine
	go func() {
		server := mockServer()
		c, err := net.ListenPacket("udp", "localhost:6677")
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()

		// Start the client
		start <- true
		packet, err := server.ReceivePacket(context.Background(), c)
		if err != nil {
			t.Error("Server error")
			return
		}
		if packet == nil {
			t.Error("nil packet")
			return
		}
		msg := packet.(*Message)
		if msg.CountArguments() != 2 {
			t.Errorf("Argument length should be 2 and is: %d\n", msg.CountArguments())
		}
		if msg.Arguments[0].(int32) != 1122 {
			t.Error("Argument should be 1122 and is: " + string(msg.Arguments[0].(int32)))
		}
		if msg.Arguments[1].(int32) != 3344 {
			t.Error("Argument should be 3344 and is: " + string(msg.Arguments[1].(int32)))
		}

		c.Close()
		finish <- true
	}()

	go func() {
		timeout := time.After(5 * time.Second)
		select {
		case <-timeout:
		case <-start:
			client := NewClient("localhost", 6677)
			msg := NewMessage("/address/test")
			msg.Append(int32(1122))
			msg.Append(int32(3344))
			time.Sleep(500 * time.Millisecond)
			client.Send(msg)
		}

		done.Done()

		select {
		case <-timeout:
		case <-finish:
		}
		done.Done()
	}()

	done.Wait()
}

func TestReadTimeout(t *testing.T) {
	start := make(chan bool)
	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()

		select {
		case <-time.After(5 * time.Second):
			t.Fatal("timed out")
		case <-start:
			client := NewClient("localhost", 6677)
			msg := NewMessage("/address/test1")
			err := client.Send(msg)
			if err != nil {
				t.Fatal(err)
			}
			time.Sleep(150 * time.Millisecond)
			msg = NewMessage("/address/test2")
			err = client.Send(msg)
			if err != nil {
				t.Fatal(err)
			}
		}
	}()

	go func() {
		defer wg.Done()

		timeout := 100 * time.Millisecond

		server := mockServer()
		c, err := net.ListenPacket("udp", "localhost:6677")
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()

		start <- true
		ctx, _ := context.WithTimeout(context.Background(), timeout)
		p, err := server.ReceivePacket(ctx, c)
		if err != nil {
			t.Errorf("server error: %v", err)
			return
		}
		if got, want := p.(*Message).Address, "/address/test1"; got != want {
			t.Errorf("wrong address; got = %s, want = %s", got, want)
			return
		}

		// Second receive should time out since client is delayed 150 milliseconds
		ctx, _ = context.WithTimeout(context.Background(), timeout)
		if _, err = server.ReceivePacket(ctx, c); err == nil {
			t.Errorf("expected error")
			return
		}

		// Next receive should get it
		ctx, _ = context.WithTimeout(context.Background(), timeout)
		p, err = server.ReceivePacket(ctx, c)
		if err != nil {
			t.Errorf("server error: %v", err)
			return
		}
		if got, want := p.(*Message).Address, "/address/test2"; got != want {
			t.Errorf("wrong address; got = %s, want = %s", got, want)
			return
		}
	}()

	wg.Wait()
}

func mockServer() *Server {
	return &Server{Addr: "localhost"}
}
