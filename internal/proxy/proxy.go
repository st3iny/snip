package proxy

import (
	"io"
	"net"
	"sync"
)

type ProxyStats struct {
	BackendToClient int64
	ClientToBackend int64
}

func Proxy(clientConn, backendConn net.Conn, peekedClientBytes io.Reader) ProxyStats {
	// TODO: error handling?

	var wg sync.WaitGroup
	wg.Add(2)

	var backendToClient int64
	var clientToBackend int64

	// backend -> client
	go func() {
		backendToClient, _ = io.Copy(clientConn, backendConn)
		clientConn.(*net.TCPConn).CloseWrite()
		wg.Done()
	}()

	// client -> backend
	go func() {
		clientToBackend, _ = io.Copy(backendConn, peekedClientBytes)
		bytes, _ := io.Copy(backendConn, clientConn)
		clientToBackend += bytes
		backendConn.(*net.TCPConn).CloseWrite()
		wg.Done()
	}()

	wg.Wait()

	return ProxyStats{
		BackendToClient: backendToClient,
		ClientToBackend: clientToBackend,
	}
}
