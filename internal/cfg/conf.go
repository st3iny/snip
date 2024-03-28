package cfg

import "snip.io/internal/backend"

type Conf struct {
	Listen string
	Backends map[string]*backend.Backend
	Frontends map[string]*backend.Backend
}

func Parse(path string) (*Conf, error) {
	snipfile, err := ParseSnipfile(path)
	if err != nil {
		return nil, err
	}

	listen := ":443"
	if snipfile.HasGlobal("listen") {
		listen = snipfile.GetGlobal("listen")
	}

	conf := &Conf{
		Listen: listen,
		Backends: make(map[string]*backend.Backend),
		Frontends: make(map[string]*backend.Backend),
	}

	for _, back := range snipfile.Backends {
		conf.Backends[back.Name] = &backend.Backend{
			UpstreamAddr:  back.Upstream,
			ProxyProtocol: back.Block.Has("proxy_protocol"),
		}
	}

	for _, frontend := range snipfile.Frontends {
		back, ok := conf.Backends[frontend.Backend]
		if !ok {
			back = &backend.Backend {
				UpstreamAddr:  frontend.Backend,
				ProxyProtocol: frontend.Block.Has("proxy_protocol"),
			}
		}

		conf.Frontends[frontend.Domain] = back
	}

	return conf, nil
}

func (conf *Conf) GetBackend(domain string) *backend.Backend {
	// TODO: match wildcards

	backend, ok := conf.Frontends[domain]
	if !ok {
		return nil
	}

	return backend
}
