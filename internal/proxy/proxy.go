package proxy

import (
	"io"
	"log"
	"net"
	"sync"
)

type ProxyStats struct {
	BackendToClient int64
	ClientToBackend int64
}

func Proxy(clientConn, backendConn net.Conn, peekedClientBytes io.Reader) ProxyStats {
	var wg sync.WaitGroup
	wg.Add(2)

	var backendToClient int64
	var clientToBackend int64

	// backend -> client
	go func() {
		var err error

		backendToClient, err = io.Copy(clientConn, backendConn)
		if err != nil {
			log.Println("Error during backend -> client communication:", err)
		}

		clientConn.(*net.TCPConn).CloseWrite()
		wg.Done()
	}()

	// client -> backend
	go func() {
		var err error

		clientToBackend, err = io.Copy(backendConn, peekedClientBytes)
		if err != nil {
			log.Println("Error during client -> backend communication (peeked bytes):", err)
		}

		bytes, err := io.Copy(backendConn, clientConn)
		clientToBackend += bytes
		if err != nil {
			log.Println("Error during client -> backend communication:", err)
		}

		backendConn.(*net.TCPConn).CloseWrite()
		wg.Done()
	}()

	wg.Wait()

	return ProxyStats{
		BackendToClient: backendToClient,
		ClientToBackend: clientToBackend,
	}
}
