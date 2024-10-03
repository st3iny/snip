package router

import (
	"fmt"
	"net"
	"strings"
	"time"

	proxyproto "github.com/pires/go-proxyproto"
)

// TODO: implement load balancing
type Backend struct {
	Name          string
	UpstreamAddr  string
	ProxyProtocol bool
}

func (b *Backend) Dial(clientAddr net.Addr) (net.Conn, error) {
	conn, err := net.Dial("tcp", b.UpstreamAddr)
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
	conn, err := net.DialTimeout("tcp", b.UpstreamAddr, timeout)
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

func sendProxyProtocolHeader(conn net.Conn, clientAddr net.Addr) error {
	clientAddrString := clientAddr.String()

	var protocol proxyproto.AddressFamilyAndProtocol
	if strings.Contains(clientAddrString, ".") {
		protocol = proxyproto.TCPv4
	} else if strings.Contains(clientAddrString, ":") {
		protocol = proxyproto.TCPv6
	} else {
		return fmt.Errorf("Unknown client address network: %s", clientAddrString)
	}

	header := &proxyproto.Header{
		Version:           1,
		Command:           proxyproto.PROXY,
		TransportProtocol: protocol,
		SourceAddr:        clientAddr,
		DestinationAddr:   conn.RemoteAddr(),
	}

	_, err := header.WriteTo(conn)
	if err != nil {
		return err
	}

	return nil
}
