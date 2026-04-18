package impl

import (
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/triasbrata/higo/routers"
)

type sFiber struct {
	Engine      *fiber.App
	middlewares []fiber.Handler
	router      fiber.Router
	mut         *sync.Mutex
	parent      *sFiber
}

// GroupWithMiddleware implements routers.Router.
func (s *sFiber) GroupWithMiddleware(prefix string, middleware []fiber.Handler, fn func(group routers.Router)) routers.Router {
	return s.group(prefix, fn, middleware)
}

// Add implements routers.Router.
func (s *sFiber) Add(method string, path string, handler fiber.Handler) routers.Router {
	s.router.Add(method, path, append(s.middlewares, handler)...)
	return s
}

// Delete implements routers.Router.
func (s *sFiber) Delete(path string, handler fiber.Handler) routers.Router {
	s.router.Delete(path, append(s.middlewares, handler)...)
	return s
}

// Get implements routers.Router.
func (s *sFiber) Get(path string, handler fiber.Handler) routers.Router {
	s.router.Get(path, append(s.middlewares, handler)...)
	return s
}

// GlobalMiddleware implements routers.Router.
func (s *sFiber) GlobalMiddleware(handler fiber.Handler) routers.Router {
	s.mut.Lock()
	defer s.mut.Unlock()
	roots := s.findRoot()
	roots.router.Use(handler)
	return s
}

// Middleware implements routers.Router.
func (s *sFiber) Middleware(handler ...fiber.Handler) routers.Router {
	s.middlewares = append(s.middlewares, handler...)
	return s
}

// Post implements routers.Router.
func (s *sFiber) Post(path string, handler fiber.Handler) routers.Router {
	s.router.Post(path, append(s.middlewares, handler)...)
	return s
}

// Put implements routers.Router.
func (s *sFiber) Put(path string, handler fiber.Handler) routers.Router {
	s.router.Put(path, append(s.middlewares, handler)...)
	return s
}

// Patch implements routers.Router.
func (s *sFiber) Patch(path string, handler fiber.Handler) routers.Router {
	s.router.Patch(path, append(s.middlewares, handler)...)
	return s
}

// Route implements routers.Router.
func (s *sFiber) Group(prefix string, fn func(router routers.Router)) routers.Router {
	return s.group(prefix, fn, []fiber.Handler{})
}

func (s *sFiber) group(prefix string, fn func(router routers.Router), middleware []fiber.Handler) routers.Router {
	s.router.Route(prefix, func(router fiber.Router) {
		fn(&sFiber{s.Engine, append(s.middlewares, middleware...), router, s.mut, s})
	})
	return s
}

// Static implements routers.Router.
func (s *sFiber) Static(prefix string, root string, config ...fiber.Static) routers.Router {
	s.router.Static(prefix, root, config...)
	return s
}

func NewEngine(app *fiber.App) routers.Router {
	return &sFiber{Engine: app, middlewares: []fiber.Handler{}, mut: &sync.Mutex{}, router: app}
}

func (s *sFiber) findRoot() *sFiber {
	if s.parent != nil {
		return s.parent.findRoot()
	}
	return s
}
