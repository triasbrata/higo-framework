package routers

import "github.com/gofiber/fiber/v2"

type Router interface {
	Add(method, path string, handler fiber.Handler) Router
	Get(path string, handler fiber.Handler) Router
	Post(path string, handler fiber.Handler) Router
	Put(path string, handler fiber.Handler) Router
	Patch(path string, handler fiber.Handler) Router
	Delete(path string, handler fiber.Handler) Router
	Static(prefix, root string, config ...fiber.Static) Router
	Middleware(handler ...fiber.Handler) Router
	Group(prefix string, fn func(group Router)) Router
	GroupWithMiddleware(prefix string, middleware []fiber.Handler, fn func(group Router)) Router
	GlobalMiddleware(handler fiber.Handler) Router
}
