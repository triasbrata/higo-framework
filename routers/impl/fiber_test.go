package impl

import (
	"context"
	"io"
	"net/http"
	"sync"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/triasbrata/higo/routers"
)

func setupFiberRouter() *sFiber {
	app := fiber.New()
	return &sFiber{
		Engine:      app,
		middlewares: []fiber.Handler{},
		mut:         &sync.Mutex{},
		router:      app,
	}
}
func TestHeadRoute(t *testing.T) {

	bodyStringRes := "Head Route"
	testCase := []struct {
		name      string
		routing   func(r routers.Router)
		expectRes string
		url       string
		method    string
	}{
		{
			name: "GET /get",
			routing: func(r routers.Router) {
				r.Get("/get", func(c *fiber.Ctx) error {
					return c.SendString(bodyStringRes)
				})
			},
			expectRes: bodyStringRes,
			url:       "/get",
			method:    http.MethodGet,
		},
		{
			name: "use GET /get",
			routing: func(r routers.Router) {
				r.Group("/api/v1", func(router routers.Router) {
					router.Get("/get", func(c *fiber.Ctx) error {
						return c.SendString(bodyStringRes)
					})
				})

			},
			expectRes: bodyStringRes,
			url:       "/api/v1/get",
			method:    http.MethodGet,
		},
		{
			name: "test with global Middleware and middleware",
			routing: func(r routers.Router) {
				r.GlobalMiddleware(func(c *fiber.Ctx) error {
					c.SetUserContext(context.WithValue(c.UserContext(), "global", "world"))
					return c.Next()
				}).Middleware(func(c *fiber.Ctx) error {
					c.SetUserContext(context.WithValue(c.UserContext(), "hello", "world"))
					// assert.NotNil(t, c)
					return c.Next()
				}).Group("/api/v1", func(router routers.Router) {
					router.Get("/get", func(c *fiber.Ctx) error {
						val := c.UserContext().Value("hello").(string)
						val2 := c.UserContext().Value("global").(string)
						return c.SendString(bodyStringRes + val + val2)
					})
				})

			},
			expectRes: bodyStringRes + "worldworld",
			url:       "/api/v1/get",
			method:    http.MethodGet,
		},
		{
			name: "test are group middleware outof group",
			routing: func(r routers.Router) {
				r.GlobalMiddleware(func(c *fiber.Ctx) error {
					c.SetUserContext(context.WithValue(c.UserContext(), "global", "world"))
					return c.Next()
				}).Group("/api/v1", func(router routers.Router) {
					router.Middleware(func(c *fiber.Ctx) error {
						c.SetUserContext(context.WithValue(c.UserContext(), "hello", "world"))
						// assert.NotNil(t, c)
						return c.Next()
					}).Get("/get", func(c *fiber.Ctx) error {
						val := c.UserContext().Value("hello").(string)
						val2 := c.UserContext().Value("global").(string)
						return c.SendString(bodyStringRes + val + val2)
					})
				}).Get("/get2", func(c *fiber.Ctx) error {
					val, safe := c.UserContext().Value("hello").(string)
					assert.Equal(t, safe, false)
					val2, safe := c.UserContext().Value("global").(string)
					assert.Equal(t, safe, true)
					return c.SendString(bodyStringRes + val + val2)
				})

			},
			expectRes: "Head Routeworld",
			url:       "/get2",
			method:    http.MethodGet,
		},
		{
			name: "test put global middleware in group",
			routing: func(r routers.Router) {
				r.Group("/api/v1", func(router routers.Router) {
					r.GlobalMiddleware(func(c *fiber.Ctx) error {
						c.SetUserContext(context.WithValue(c.UserContext(), "global", "world"))
						return c.Next()
					})
					router.Middleware(func(c *fiber.Ctx) error {
						c.SetUserContext(context.WithValue(c.UserContext(), "hello", "world"))
						// assert.NotNil(t, c)
						return c.Next()
					}).Get("/get", func(c *fiber.Ctx) error {
						val := c.UserContext().Value("hello").(string)
						val2 := c.UserContext().Value("global").(string)
						return c.SendString(bodyStringRes + val + val2)
					})
				}).Get("/get2", func(c *fiber.Ctx) error {
					val, safe := c.UserContext().Value("hello").(string)
					assert.Equal(t, safe, false)
					val2, safe := c.UserContext().Value("global").(string)
					assert.Equal(t, safe, true)
					return c.SendString(bodyStringRes + val + val2)
				})

			},
			expectRes: "Head Routeworld",
			url:       "/get2",
			method:    http.MethodGet,
		},
		{
			name: "test put global middleware in group use router group",
			routing: func(r routers.Router) {
				r.Group("/api/v1", func(router routers.Router) {
					router.GlobalMiddleware(func(c *fiber.Ctx) error {
						c.SetUserContext(context.WithValue(c.UserContext(), "global", "world"))
						return c.Next()
					})
					router.Middleware(func(c *fiber.Ctx) error {
						c.SetUserContext(context.WithValue(c.UserContext(), "hello", "world"))
						// assert.NotNil(t, c)
						return c.Next()
					}).Get("/get", func(c *fiber.Ctx) error {
						val := c.UserContext().Value("hello").(string)
						val2 := c.UserContext().Value("global").(string)
						return c.SendString(bodyStringRes + val + val2)
					})
				}).Get("/get2", func(c *fiber.Ctx) error {
					val, safe := c.UserContext().Value("hello").(string)
					assert.Equal(t, safe, false)
					val2, safe := c.UserContext().Value("global").(string)
					assert.Equal(t, safe, true)
					return c.SendString(bodyStringRes + val + val2)
				})
			},
			expectRes: "Head Routeworld",
			url:       "/get2",
			method:    http.MethodGet,
		},
		{
			name: "test group middleware",
			routing: func(r routers.Router) {
				r.GroupWithMiddleware("/api/v1", []fiber.Handler{}, func(router routers.Router) {
					router.GlobalMiddleware(func(c *fiber.Ctx) error {
						c.SetUserContext(context.WithValue(c.UserContext(), "global", "world"))
						return c.Next()
					}).Middleware(func(c *fiber.Ctx) error {
						c.SetUserContext(context.WithValue(c.UserContext(), "hello", "world"))
						// assert.NotNil(t, c)
						return c.Next()
					}).Get("/get", func(c *fiber.Ctx) error {
						val := c.UserContext().Value("hello").(string)
						val2 := c.UserContext().Value("global").(string)
						return c.SendString(bodyStringRes + val + val2)
					})
				})
			},
			expectRes: "Head Routeworld",
			url:       "/get2",
			method:    http.MethodGet,
		},
		{
			name: "POST /post",
			routing: func(r routers.Router) {
				r.Post("/post", func(c *fiber.Ctx) error {
					return c.SendString(bodyStringRes)
				})
			},
			expectRes: bodyStringRes,
			url:       "/post",
			method:    http.MethodPost,
		},
		{
			name: "PUT /put",
			routing: func(r routers.Router) {
				r.Put("/put", func(c *fiber.Ctx) error {
					return c.SendString(bodyStringRes)
				})
			},
			expectRes: bodyStringRes,
			url:       "/put",
			method:    http.MethodPut,
		},
		{
			name: "DELETE /delete",
			routing: func(r routers.Router) {
				r.Delete("/delete", func(c *fiber.Ctx) error {
					return c.SendString(bodyStringRes)
				})
			},
			expectRes: bodyStringRes,
			url:       "/delete",
			method:    http.MethodDelete,
		},
		{
			name: "PATCH /patch",
			routing: func(r routers.Router) {
				r.Patch("/patch", func(c *fiber.Ctx) error {
					return c.SendString(bodyStringRes)
				})
			},
			expectRes: bodyStringRes,
			url:       "/patch",
			method:    http.MethodPatch,
		},
		{
			name: "OPTIONS /options",
			routing: func(r routers.Router) {
				r.Add(http.MethodOptions, "/options", func(c *fiber.Ctx) error {
					return c.SendString(bodyStringRes)
				})
			},
			expectRes: bodyStringRes,
			url:       "/options",
			method:    http.MethodOptions,
		},
		{
			name: "HEAD /head",
			routing: func(r routers.Router) {
				r.Add(http.MethodHead, "/head", func(c *fiber.Ctx) error {
					return c.SendString(bodyStringRes)
				})
			},
			expectRes: "",
			url:       "/head",
			method:    http.MethodHead,
		},
		{
			name: "CONNECT /connect",
			routing: func(r routers.Router) {
				r.Add(http.MethodConnect, "/connect", func(c *fiber.Ctx) error {
					return c.SendString(bodyStringRes)
				})
			},
			expectRes: bodyStringRes,
			url:       "/connect",
			method:    http.MethodConnect,
		},
		{
			name: "TRACE /trace",
			routing: func(r routers.Router) {
				r.Add(http.MethodTrace, "/trace", func(c *fiber.Ctx) error {
					return c.SendString(bodyStringRes)
				})
			},
			expectRes: bodyStringRes,
			url:       "/trace",
			method:    http.MethodTrace,
		},
	}
	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			r := setupFiberRouter()
			tc.routing(r)
			// bytes, _ := json.Marshal(r.Engine.Stack())
			// fmt.Printf("r.Engine.Stack(): %s\n", bytes)
			newVar, err := http.NewRequest(tc.method, tc.url, http.NoBody)
			assert.NoError(t, err)
			res, err := r.Engine.Test(newVar)
			assert.NoError(t, err)
			resByte, err := io.ReadAll(res.Body)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectRes, string(resByte))
		})
	}

}
