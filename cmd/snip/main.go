package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"snip.io/internal/cfg"
	"snip.io/internal/proxy"
	"snip.io/internal/sni"
)

func main() {
	pid := os.Getpid()
	log.Println("Snip running with pid", pid)
	os.WriteFile("/var/tmp/snip.pid", []byte(fmt.Sprint(pid)), 0644)

	confPath := "/etc/snip/config.toml"
	if len(os.Args) >= 2 {
		confPath = os.Args[1]
	}

	conf, err := cfg.Parse(confPath)
	if err != nil {
		log.Fatal(err)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGUSR1)

	quit := make(chan bool, 1)
	confChannel := make(chan *cfg.Conf, 1)

	go func() {
		for {
			<-sigs
			log.Println("Reloading config")

			conf, err := cfg.Parse(confPath)
			if err != nil {
				log.Println("Keeping old config:", err)
				continue
			}

			quit <- true
			confChannel <- conf
		}
	}()

	for {
		server := server{
			conf: conf,
			quit: quit,
		}
		server.run()
		conf = <-confChannel
	}
}

type server struct {
	conf *cfg.Conf
	quit chan bool
}

func (s *server) run() {
	l, err := net.Listen("tcp", s.conf.Listen)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Listening on", s.conf.Listen)

	shuttingDown := false
	go func() {
		<-s.quit
		shuttingDown = true
		l.Close()
	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			if !shuttingDown {
				log.Println("Accepting connection failed:", err)
			}
			break
		}

		go s.handleConnection(conn)
	}

	log.Println("Server shutting down")
}

func (s *server) handleConnection(clientConn net.Conn) {
	defer clientConn.Close()

	if err := clientConn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
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

	backend := s.conf.Router.GetBackend(clientHello.ServerName)
	if backend == nil {
		log.Println("No backend for server name:", clientHello.ServerName)
		return
	}

	if len(backend.UpstreamAddrs) == 0 {
		log.Printf("Backend %s has no upstreams\n", backend.Name)
		return
	}

	var backendConn net.Conn
	if backend.ConnectTimeout > 0 {
		timeout := time.Duration(backend.ConnectTimeout) * time.Second
		backendConn, err = backend.DialTimeout(clientConn.RemoteAddr(), timeout)
	} else {
		backendConn, err = backend.Dial(clientConn.RemoteAddr())
	}
	if err != nil {
		log.Println(err)
		return
	}
	defer backendConn.Close()

	stats := proxy.Proxy(clientConn, backendConn, peekedClientBytes)
	log.Printf("Proxied %s -> %s (from client %d, to client %d bytes)\n", clientConn.RemoteAddr(), backendConn.RemoteAddr(), stats.ClientToBackend, stats.BackendToClient)
}
