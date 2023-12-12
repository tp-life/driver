package http

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

type Option struct {
	Port int `json:"port"`
}

type RuntimeOption struct {
	Pprof      bool
	IsRelease  bool
	AppName    string
	Middleware []fiber.Handler
}

func DefaultRuntimeOptions() RuntimeOption {
	return RuntimeOption{
		Middleware: []fiber.Handler{
			cors.New(),
			limiter.New(limiter.Config{Max: 100, Expiration: time.Second}),
		},
	}
}
