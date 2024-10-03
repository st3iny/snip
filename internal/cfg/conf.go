package cfg

import (
	"fmt"
	"log"
	"os"

	"github.com/BurntSushi/toml"
	"snip.io/internal/router"
)

const defaultListen = ":443"

type Conf struct {
	Listen string
	Router router.Router
}

func Parse(path string) (*Conf, error) {
	rawConfigBytes, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		log.Printf("Using empty config file (%s does not exit)\n", path)
		emptyConf := &Conf{
			Listen: defaultListen,
			Router: router.Router{
				Frontends: []router.Frontend{},
				Backends:  []router.Backend{},
			},
		}
		return emptyConf, nil
	} else if err != nil {
		return nil, err
	}

	var rawConfig rawConf
	err = toml.Unmarshal(rawConfigBytes, &rawConfig)
	if err != nil {
		return nil, err
	}

	if rawConfig.Listen == "" {
		rawConfig.Listen = defaultListen
	}

	if len(rawConfig.Frontends) == 0 {
		log.Println("Config file does not contain any frontends")
	}

	var backends []router.Backend
	for _, backend := range rawConfig.Backends {
		backends = append(backends, router.Backend{
			Name:          backend.Name,
			UpstreamAddrs: backend.Upstreams,
			ProxyProtocol: backend.ProxyProtocol,
		})
	}

	var frontends []router.Frontend
	for i, frontend := range rawConfig.Frontends {
		backend := getBackend(frontend.Backend, backends)
		if backend == nil {
			backend = &router.Backend{
				Name:          frontend.Backend,
				UpstreamAddrs: []string{frontend.Backend},
				ProxyProtocol: false,
			}
		}

		var matchers []router.DomainMatcher
		for j, match := range frontend.Match {
			matcher, err := router.ParseMatcher(match)
			if err != nil {
				return nil, fmt.Errorf("Failed to parse matcher %d of fronted %d: %s", j, i, err)
			}

			matchers = append(matchers, matcher)
		}

		frontends = append(frontends, router.Frontend{
			Match:   matchers,
			Backend: backend,
		})
	}

	log.Println("Using config file at", path)
	config := &Conf{
		Listen: rawConfig.Listen,
		Router: router.Router{
			Frontends: frontends,
			Backends:  backends,
		},
	}
	return config, nil
}

func getBackend(name string, backends []router.Backend) *router.Backend {
	for i, backend := range backends {
		if backend.Name == name {
			return &backends[i]
		}
	}

	return nil
}

type rawFrontend struct {
	Match   []string `toml:"match"`
	Backend string   `toml:"backend"`
}

type rawBackend struct {
	Name          string   `toml:"name"`
	Upstreams     []string `toml:"upstreams"`
	ProxyProtocol bool     `toml:"proxy_protocol"`
}

type rawConf struct {
	Listen    string        `toml:"listen"`
	Frontends []rawFrontend `toml:"frontend"`
	Backends  []rawBackend  `toml:"backend"`
}
