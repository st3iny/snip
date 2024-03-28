package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"snip.io/internal/cfg"
	"snip.io/internal/proxy"
	"snip.io/internal/sni"
)

func main() {
	confPath := "/etc/snip/Snipfile"
	if len(os.Args) >= 2 {
		confPath = os.Args[1]
	}
	log.Println("Using config at", confPath)

	conf, err := cfg.Parse(confPath)
	if err != nil {
		log.Fatal(err)
	}

	l, err := net.Listen("tcp", conf.Listen)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Listening on", conf.Listen)

	server := server {
		conf: conf,
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGUSR1)

	go func() {
		for {
			<- sigs
			log.Println("Reloading config")

			conf, err := cfg.Parse(confPath)
			if err != nil {
				log.Println("Keeping old config:", err)
				continue
			}

			server.confMutex.Lock()
			server.conf = conf
			server.confMutex.Unlock()
		}
	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		go server.handleConnection(conn)
	}
}

type server struct {
	conf *cfg.Conf
	confMutex sync.RWMutex
}

func (s *server) handleConnection(clientConn net.Conn) {
	defer clientConn.Close()

	if err := clientConn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		log.Println(err)
		return
	}

	clientHello, peekedClientBytes, err := sni.PeekClientHello(clientConn)
	if err != nil {
		log.Println(err)
		return
	}

	if err := clientConn.SetReadDeadline(time.Time{}); err != nil {
		log.Println(err)
		return
	}

	s.confMutex.RLock()
	backend := s.conf.GetBackend(clientHello.ServerName)
	s.confMutex.RUnlock()
	if backend == nil {
		log.Println("No backend for server name:", clientHello.ServerName)
		return
	}

	backendConn, err := backend.DialTimeout(clientConn.RemoteAddr(), 5 * time.Second)
	if err != nil {
		log.Println(err)
		return
	}
	defer backendConn.Close()

	log.Printf("Proxying %s -> %s\n", clientConn.RemoteAddr(), backendConn.RemoteAddr())
	proxy.Proxy(clientConn, backendConn, peekedClientBytes)
}

