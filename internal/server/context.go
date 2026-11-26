package server

import (
	"slices"

	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
)

const fallbackMethod = "*"

// routingContext permite que métodos no registrados recorran los middleware y
// terminen en los mismos 404/405 JSON que antes. Fiber devuelve 501 antes del
// router para esos métodos. El método original sigue visible en logs y handlers.
type routingContext struct {
	*fiber.DefaultCtx
	originalMethod string
}

func (c *routingContext) Reset(ctx *fasthttp.RequestCtx) {
	c.DefaultCtx.Reset(ctx)
	c.originalMethod = ""
	method := c.DefaultCtx.Method()
	if !slices.Contains(c.App().Config().RequestMethods, method) {
		c.originalMethod = method
		c.DefaultCtx.Method(fallbackMethod)
	}
}

func (c *routingContext) Method(override ...string) string {
	if len(override) == 0 && c.originalMethod != "" {
		return c.originalMethod
	}
	return c.DefaultCtx.Method(override...)
}
