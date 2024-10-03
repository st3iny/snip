package router

type Router struct {
	Frontends []Frontend
	Backends  []Backend
}

func (router *Router) GetBackend(domain string) *Backend {
	frontend := router.matchFrontend(domain)
	if frontend == nil {
		return nil
	}

	return frontend.Backend
}

func (router *Router) matchFrontend(domain string) *Frontend {
	for i, frontend := range router.Frontends {
		if frontend.MatchesDomain(domain) {
			return &router.Frontends[i]
		}
	}

	return nil
}
