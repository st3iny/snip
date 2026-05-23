package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"snip.io/internal/cfg"
	"snip.io/internal/router"
)

func TestSnip(t *testing.T) {
	t.Run("catchall", func(t *testing.T) {
		payload := []byte("Hello, World!")

		quit := make(chan bool, 1)

		srvCalled := 0
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			assert.Equal(t, payload, body)

			w.WriteHeader(http.StatusOK)
			_, err := w.Write(payload)
			assert.NoError(t, err)

			srvCalled++
			quit <- true
		}))
		defer srv.Close()

		conf := &cfg.Conf{
			Listen: fmt.Sprintf("127.0.0.1:%d", 49152+rand.Intn(65535-49152)),
			Router: router.Router{
				Frontends: []router.Frontend{
					{
						Match: []router.DomainMatcher{matcher(t, "*")},
						Backend: &router.Backend{
							UpstreamAddrs: []string{strings.TrimPrefix(srv.URL, "https://")},
						},
					},
				},
			},
		}
		server := server{
			conf: conf,
			quit: quit,
		}
		go server.run()

		certs := x509.NewCertPool()
		certs.AddCert(srv.Certificate())

		client := http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:    certs,
					ServerName: "example.com",
				},
			},
		}

		res, err := client.Post(fmt.Sprintf("https://%s", conf.Listen), "application/octet-stream", bytes.NewBuffer(payload))
		require.NoError(t, err)
		defer res.Body.Close()

		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)

		assert.Equal(t, payload, body)
		assert.Equal(t, 1, srvCalled)
	})
	t.Run("fqdn", func(t *testing.T) {
		payload := []byte("Hello, World!")

		quit := make(chan bool, 1)

		srvCalled := 0
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			assert.Equal(t, payload, body)

			w.WriteHeader(http.StatusOK)
			_, err := w.Write(payload)
			assert.NoError(t, err)

			srvCalled++
			quit <- true
		}))
		defer srv.Close()

		conf := &cfg.Conf{
			Listen: fmt.Sprintf("127.0.0.1:%d", 49152+rand.Intn(65535-49152)),
			Router: router.Router{
				Frontends: []router.Frontend{
					{
						Match: []router.DomainMatcher{matcher(t, "example.com")},
						Backend: &router.Backend{
							UpstreamAddrs: []string{strings.TrimPrefix(srv.URL, "https://")},
						},
					},
				},
			},
		}
		server := server{
			conf: conf,
			quit: quit,
		}
		go server.run()

		certs := x509.NewCertPool()
		certs.AddCert(srv.Certificate())

		client := http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:    certs,
					ServerName: "example.com",
				},
			},
		}

		res, err := client.Post(fmt.Sprintf("https://%s", conf.Listen), "application/octet-stream", bytes.NewBuffer(payload))
		require.NoError(t, err)
		defer res.Body.Close()

		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)

		assert.Equal(t, payload, body)
		assert.Equal(t, 1, srvCalled)
	})
}

func TestSnip_HandleConnection(t *testing.T) {
	t.Run("no match", func(t *testing.T) {
		srvCalled := 0
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			srvCalled++
		}))
		defer srv.Close()

		conf := &cfg.Conf{
			Router: router.Router{
				Frontends: []router.Frontend{
					{
						Match: []router.DomainMatcher{matcher(t, "example.com")},
						Backend: &router.Backend{
							UpstreamAddrs: []string{strings.TrimPrefix(srv.URL, "https://")},
						},
					},
				},
			},
		}
		server := server{
			conf: conf,
		}

		mockServer, mockClient := net.Pipe()
		defer mockServer.Close()
		defer mockClient.Close()

		client := tls.Client(mockClient, &tls.Config{
			ServerName:         "test.example.com",
			InsecureSkipVerify: true,
		})

		clientDone := make(chan error, 1)
		go func() {
			clientDone <- client.Handshake()
		}()

		serverDone := make(chan error, 1)
		go func() {
			serverDone <- server.handleConnection(mockServer)
		}()

		assert.ErrorIs(t, <-clientDone, io.EOF)
		assert.ErrorContains(t, <-serverDone, "No backend for server name: test.example.com")

		assert.Equal(t, 0, srvCalled)
	})
	t.Run("client sends no sni", func(t *testing.T) {
		srvCalled := 0
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			srvCalled++
		}))
		defer srv.Close()

		conf := &cfg.Conf{
			Listen: "127.0.0.1:50000",
			Router: router.Router{
				Frontends: []router.Frontend{
					{
						Match: []router.DomainMatcher{matcher(t, "counter.com")},
						Backend: &router.Backend{
							UpstreamAddrs: []string{strings.TrimPrefix(srv.URL, "https://")},
						},
					},
				},
			},
		}
		server := server{
			conf: conf,
		}

		mockServer, mockClient := net.Pipe()
		defer mockServer.Close()
		defer mockClient.Close()

		client := tls.Client(mockClient, &tls.Config{
			ServerName:         "",
			InsecureSkipVerify: true,
		})

		clientDone := make(chan error, 1)
		go func() {
			clientDone <- client.Handshake()
		}()

		serverDone := make(chan error, 1)
		go func() {
			serverDone <- server.handleConnection(mockServer)
		}()

		assert.Equal(t, 0, srvCalled)

		assert.ErrorIs(t, <-clientDone, io.EOF)
		assert.ErrorContains(t, <-serverDone, "Client sent no SNI")
	})
}

func matcher(t *testing.T, match string) router.DomainMatcher {
	t.Helper()

	matcher, err := router.ParseMatcher(match)
	require.NoError(t, err)
	return matcher
}
