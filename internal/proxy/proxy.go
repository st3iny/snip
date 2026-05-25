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

func Proxy(clientConn, backendConn net.Conn, peekedClientBytes []byte) ProxyStats {
	var wg sync.WaitGroup

	var backendToClient int64
	var clientToBackend int64

	// backend -> client
	wg.Go(func() {
		defer clientConn.(*net.TCPConn).CloseWrite()

		var err error
		backendToClient, err = io.Copy(clientConn, backendConn)
		if err != nil {
			log.Println("Error during backend -> client communication:", err)
		}
	})

	// client -> backend
	wg.Go(func() {
		defer backendConn.(*net.TCPConn).CloseWrite()

		peekedByteCount, err := backendConn.Write(peekedClientBytes)
		clientToBackend = int64(peekedByteCount)
		if err != nil {
			log.Println("Error during client -> backend communication (peeked bytes):", err)
		}

		remainingByteCount, err := io.Copy(backendConn, clientConn)
		clientToBackend += remainingByteCount
		if err != nil {
			log.Println("Error during client -> backend communication:", err)
		}
	})

	wg.Wait()

	return ProxyStats{
		BackendToClient: backendToClient,
		ClientToBackend: clientToBackend,
	}
}
