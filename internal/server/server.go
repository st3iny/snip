package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"snip.io/internal/cfg"
	"snip.io/internal/proxy"
	"snip.io/internal/sni"
)

var wg sync.WaitGroup

type Server struct {
	conf *cfg.Conf
}

func New(conf *cfg.Conf) *Server {
	return &Server{
		conf: conf,
	}
}

func (s *Server) Run(ctx context.Context) {
	l, err := net.Listen("tcp", s.conf.Listen)
	if err != nil {
		log.Fatal("Failed to listen on socket:", err)
	}
	defer l.Close()

	log.Println("Listening on", s.conf.Listen)

	context.AfterFunc(ctx, func() {
		l.Close()
	})

	for {
		conn, err := l.Accept()
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				log.Println("Accepting connection failed:", err)
			}
			break
		}

		wg.Go(func() {
			err := s.handleConnection(conn)
			if err != nil {
				log.Printf("Failed to handle connection from %v: %v\n", conn.RemoteAddr(), err)
			}
		})
	}
}

func (s *Server) handleConnection(clientConn net.Conn) error {
	defer clientConn.Close()

	if err := clientConn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
		return err
	}

	clientHello, peekedClientBytes, err := sni.PeekClientHello(clientConn)
	if err != nil {
		return err
	}

	if clientHello.ServerName == "" {
		return errors.New("Client sent no SNI")
	}

	if err := clientConn.SetReadDeadline(time.Time{}); err != nil {
		return err
	}

	backend := s.conf.Router.GetBackend(clientHello.ServerName)
	if backend == nil {
		return fmt.Errorf("No backend for server name: %s", clientHello.ServerName)
	}

	if len(backend.UpstreamAddrs) == 0 {
		return fmt.Errorf("Backend %s has no upstreams", backend.Name)
	}

	var backendConn net.Conn
	if backend.ConnectTimeout > 0 {
		timeout := time.Duration(backend.ConnectTimeout) * time.Second
		backendConn, err = backend.DialTimeout(clientConn.RemoteAddr(), timeout)
	} else {
		backendConn, err = backend.Dial(clientConn.RemoteAddr())
	}
	if err != nil {
		return err
	}
	defer backendConn.Close()

	stats := proxy.Proxy(clientConn, backendConn, peekedClientBytes)
	log.Printf("Proxied %s -> %s (from client %d, to client %d bytes)\n", clientConn.RemoteAddr(), backendConn.RemoteAddr(), stats.ClientToBackend, stats.BackendToClient)
	return nil
}

func WaitForConnections() {
	wg.Wait()
}
