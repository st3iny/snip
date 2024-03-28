package proxy

import (
	"io"
	"net"
	"sync"
)

func Proxy(clientConn, backendConn net.Conn, peekedClientBytes io.Reader) {
	// TODO: error handling?

	var wg sync.WaitGroup
	wg.Add(2)

	// backend -> client
	go func() {
		io.Copy(clientConn, backendConn)
		clientConn.(*net.TCPConn).CloseWrite()
		wg.Done()
	}()

	// client -> backend
	go func() {
		io.Copy(backendConn, peekedClientBytes)
		io.Copy(backendConn, clientConn)
		backendConn.(*net.TCPConn).CloseWrite()
		wg.Done()
	}()

	wg.Wait()
}
