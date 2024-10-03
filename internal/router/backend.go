package router

import (
	"math/rand"
	"net"
	"time"

	proxyproto "github.com/pires/go-proxyproto"
)

type Backend struct {
	Name          string
	UpstreamAddrs []string
	ProxyProtocol bool
}

func (b *Backend) Dial(clientAddr net.Addr) (net.Conn, error) {
	upstreamAddr := b.balance()
	conn, err := net.Dial("tcp", upstreamAddr)
	if err != nil {
		return nil, err
	}

	if !b.ProxyProtocol {
		return conn, nil
	}

	err = sendProxyProtocolHeader(conn, clientAddr)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (b *Backend) DialTimeout(clientAddr net.Addr, timeout time.Duration) (net.Conn, error) {
	upstreamAddr := b.balance()
	conn, err := net.DialTimeout("tcp", upstreamAddr, timeout)
	if err != nil {
		return nil, err
	}

	if !b.ProxyProtocol {
		return conn, nil
	}

	err = sendProxyProtocolHeader(conn, clientAddr)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (b *Backend) balance() string {
	upstreamLen := len(b.UpstreamAddrs)
	if upstreamLen == 1 {
		return b.UpstreamAddrs[0]
	}

	return b.UpstreamAddrs[rand.Intn(upstreamLen)]
}

func sendProxyProtocolHeader(conn net.Conn, clientAddr net.Addr) error {
	header := proxyproto.HeaderProxyFromAddrs(2, clientAddr, conn.RemoteAddr())
	_, err := header.WriteTo(conn)
	if err != nil {
		return err
	}

	return nil
}
