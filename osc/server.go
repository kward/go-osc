package osc

// Server represents an OSC server. The server listens on Address and Port for
import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

// Handler is an interface for message handlers. Every handler implementation
// for an OSC message must implement this interface.
type Handler interface {
	HandleMessage(msg *Message)
}

// HandlerFunc implements the Handler interface. Type definition for an OSC
// handler function.
type HandlerFunc func(msg *Message)

// HandleMessage calls itself with the given OSC Message. Implements the
// Handler interface.
func (f HandlerFunc) HandleMessage(msg *Message) {
	f(msg)
}

// incoming OSC packets and bundles.
type Server struct {
	opts       *serverOptions
	dispatcher *OSCDispatcher

	Addr string
}

func NewServer(addr string, opts ...func(*serverOptions) error) (*Server, error) {
	o := &serverOptions{}
	o.setReadTimeout(1 * time.Second)
	for _, opt := range opts {
		if err := opt(o); err != nil {
			return nil, err
		}
	}
	s := &Server{opts: o, Addr: addr}
	s.dispatcher = NewOSCDispatcher()
	return s, nil
}

type serverOptions struct {
	readTimeout time.Duration
}

func ServerReadTimeout(v time.Duration) func(*serverOptions) error {
	return func(o *serverOptions) error { return o.setReadTimeout(v) }
}

func (o *serverOptions) setReadTimeout(v time.Duration) error {
	o.readTimeout = v
	return nil
}

// Handle registers a new message handler function for an OSC address. The
// handler is the function called for incoming OscMessages that match 'address'.
func (s *Server) Handle(addr string, handler HandlerFunc) error {
	return s.dispatcher.AddMsgHandler(addr, handler)
}

// ListenAndServe retrieves incoming OSC packets and dispatches the retrieved
// OSC packets.
func (s *Server) ListenAndServe() error {
	ln, err := net.ListenPacket("udp", s.Addr)
	if err != nil {
		return err
	}
	return s.Serve(context.Background(), ln)
}

// Serve retrieves incoming OSC packets from the given connection and dispatches
// retrieved OSC packets. If something goes wrong an error is returned.
func (s *Server) Serve(ctx context.Context, c net.PacketConn) error {
	var tempDelay time.Duration
	for {
		msg, err := s.ReceivePacket(ctx, c)
		if err != nil {
			// Attempt exponential back-off during temporary network problems.
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				time.Sleep(tempDelay)
				continue // Try again.
			}
			return err // Error is not temporary.
		}
		tempDelay = 0
		go s.dispatcher.Dispatch(msg)
	}

	return nil
}

// ReceivePacket listens for incoming OSC packets and returns the packet and
// client address if one is received.
func (s *Server) ReceivePacket(ctx context.Context, c net.PacketConn) (Packet, error) {
	if deadline, ok := ctx.Deadline(); ok {
		if err := c.SetReadDeadline(deadline); err != nil {
			return nil, err
		}
	}

	go func() {
		select {
		// case <-time.After(200 * time.Millisecond):
		// 	log.Println("Overslept.")
		case <-ctx.Done():
			log.Println(ctx.Err())
		}
	}()

	data := make([]byte, 65535)
	n, addr, err := c.ReadFrom(data)
	if err != nil {
		return nil, err
	}

	var start int
	pkt, err := readPacket(bufio.NewReader(bytes.NewBuffer(data)), &start, n)
	if err != nil {
		return nil, err
	}
	pkt.SetAddr(addr)
	return pkt, nil
}

// Dispatcher is an interface for an OSC message dispatcher. A dispatcher is
// responsible for dispatching received OSC messages.
type Dispatcher interface {
	// Dispatch accepts a packet to dispatch.
	Dispatch(packet Packet)
}

// OSCDispatcher is a dispatcher for OSC packets. It handles the dispatching of
// received OSC packets.
type OSCDispatcher struct {
	handlers map[string]Handler
}

// Verify that interfaces are implemented properly.
var _ Dispatcher = new(OSCDispatcher)

// NewOSCDispatcher returns an OSCDispatcher.
func NewOSCDispatcher() *OSCDispatcher {
	return &OSCDispatcher{handlers: make(map[string]Handler)}
}

// AddMsgHandler adds a new message handler for the given OSC address.
func (d *OSCDispatcher) AddMsgHandler(addr string, handler HandlerFunc) error {
	for _, chr := range "*?,[]{}# " {
		if strings.Contains(addr, fmt.Sprintf("%c", chr)) {
			return fmt.Errorf("OSC Address string may not contain any characters in %q\n", chr)
		}
	}

	if addressExists(addr, d.handlers) {
		return fmt.Errorf("OSC address %q exists already", addr)
	}

	d.handlers[addr] = handler
	return nil
}

// Dispatch dispatches OSC packets. Implements the Dispatcher interface.
func (d *OSCDispatcher) Dispatch(pkt Packet) {
	switch pkt.(type) {
	default:
		return

	case *Message:
		msg, _ := pkt.(*Message)
		for addr, handler := range d.handlers {
			if msg.Match(addr) {
				handler.HandleMessage(msg)
			}
		}

	case *Bundle:
		bundle, _ := pkt.(*Bundle)
		timer := time.NewTimer(bundle.Timetag.ExpiresIn())

		go func() {
			<-timer.C
			for _, message := range bundle.Messages {
				for address, handler := range d.handlers {
					if message.Match(address) {
						handler.HandleMessage(message)
					}
				}
			}

			// Process all bundles
			for _, b := range bundle.Bundles {
				d.Dispatch(b)
			}
		}()
	}
}

// existsAddress returns true if the OSC address `addr` is found in `handlers`.
func addressExists(addr string, handlers map[string]Handler) bool {
	for h := range handlers {
		if h == addr {
			return true
		}
	}
	return false
}
