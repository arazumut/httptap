package main

import (
	"fmt"
	"net"
	"strings"
	"sync"
)

// mux dispatches TCP connections to listeners according to patterns
type mux struct {
	listenerMu sync.Mutex
	listeners  []*tcpListener

	handlerMu sync.Mutex
	handlers  []*udpMuxEntry
}

// Listen returns a net.Listener that intercepts connections according to a filter pattern.
//
// Pattern can a hostname, a :port, a hostname:port, or "*" for everything". For example:
//   - "example.com"
//   - "example.com:80"
//   - ":80"
//   - "*"
//
// Later this will be like net.Listen
func (s *mux) Listen(pattern string) net.Listener {
	s.listenerMu.Lock()
	defer s.listenerMu.Unlock()

	listener := tcpListener{pattern: pattern, connections: make(chan net.Conn, 64)}
	s.listeners = append(s.listeners, &listener)
	return &listener
}

// HandleTCP calls the handler each time a new connection is intercepted mattching the
// given filter pattern.
//
// Pattern can a hostname, a :port, a hostname:port, or "*" for everything". For example:
//   - "example.com"
//   - "example.com:80"
//   - ":80"
//   - "*"
func (s *mux) HandleTCP(pattern string, handler tcpHandlerFunc) {
	l := s.Listen(pattern)
	go func() {
		defer l.Close()
		for {
			conn, err := l.Accept()
			if err != nil {
				verbosef("accept returned errror: %v, exiting HandleFunc(%v)", err, pattern)
				return
			}

			go handler(conn)
		}
	}()
}

// HandleUDP registers a handler for UDP packets according to destination IP and/or por
//
// Pattern can a hostname, a port, a hostname:port, or "*" for everything". Ports are prepended
// with colons. Valid patterns are:
//   - "example.com"
//   - "example.com:80"
//   - ":80"
//   - "*"
//
// Later this will be like net.Listen
func (s *mux) HandleUDP(pattern string, handler udpHandlerFunc) {
	s.handlerMu.Lock()
	defer s.handlerMu.Unlock()

	s.handlers = append(s.handlers, &udpMuxEntry{pattern: pattern, handler: handler})
}

type tcpHandlerFunc func(conn net.Conn)

// match a listen pattern to an address string of the form HOST:PORT
func patternMatches(pattern, hostport string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasPrefix(pattern, ":") && strings.HasSuffix(hostport, pattern) {
		return true
	}
	return false
}

// notifyListeners is called when a new stream is created. It finds the first listener
// that will accept the given stream. It never blocks.
func (s *mux) notifyTCP(stream net.Conn) {
	s.listenerMu.Lock()
	defer s.listenerMu.Unlock()

	for _, listener := range s.listeners {
		if patternMatches(listener.pattern, stream.LocalAddr().String()) {
			listener.connections <- stream
			return
		}
	}

	verbosef("nobody listening for tcp to %v, dropping", stream.LocalAddr())
}

// notifyUDP is called when a new packet arrives. It finds the first handler
// with a pattern that matches the packet and delivers the packet to it
func (s *mux) notifyUDP(w udpResponder, packet *udpPacket) {
	s.handlerMu.Lock()
	defer s.handlerMu.Unlock()

	for _, entry := range s.handlers {
		if entry.pattern == "*" || entry.pattern == fmt.Sprintf(":%d", packet.udpheader.DstPort) {
			entry.handler(w, packet)
			return
		}
	}

	verbosef("nobody listening for udp to %v:%v, dropping!", packet.ipv4header.DstIP, packet.udpheader.DstPort)
}

// udpResponder is the interface for writing back UDP packets
type udpResponder interface {
	// write a UDP packet back to the subprocess
	Write(payload []byte) (n int, err error)

	// set the source IP in the header for UDP packets sent to Write()
	SetSourceIP(ip net.IP)

	// set the source port in the header for UDP packets sent to Write()
	SetSourcePort(port uint16)

	// set the destination IP in the header for UDP packets sent to Write()
	SetDestIP(ip net.IP)

	// set the destination port in the header for UDP packets sent to Write()
	SetDestPort(port uint16)
}
