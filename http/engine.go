package http

import (
	"fmt"
	"strconv"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

type Engine struct {
	*fiber.App
}

func NewEngine(opts ...RuntimeOption) *Engine {
	var ops RuntimeOption
	if len(opts) > 0 {
		ops = opts[0]
	}
	app := fiber.New(fiber.Config{
		AppName:     ops.AppName,
		JSONEncoder: sonic.Marshal,
		JSONDecoder: sonic.Unmarshal,
		Views:       ops.Views,
		ViewsLayout: ops.ViewsLayout,
	})
	for _, v := range ops.Middleware {
		app.Use(v)
	}
	if ops.Pprof {
		app.Use(pprof.New())
	}
	app.Use(recover.New())
	app.Use(requestid.New())
	if !ops.IsRelease {
		app.Use(logger.New(logger.Config{
			Format: "${pid} ${locals:requestid} ${status} - ${method} ${path}​\n",
		}))
	}

	return &Engine{App: app}
}

func (e *Engine) Serve(opt Option) {
	err := e.App.Listen(":" + strconv.Itoa(int(opt.Port)))
	if err != nil {
		panic(fmt.Sprintf("%d: http server start error: %+v", opt.Port, err))
	}

}

func (e *Engine) Quit() {
	e.App.Shutdown()
}

func (e *Engine) Use(args ...interface{}) {
	e.App.Use(args...)
}

func (e *Engine) UseMiddleware(args ...interface{}) fiber.Router {
	return e.App.Use(args...)
}

func (e *Engine) Get(path string, handlers ...func(*fiber.Ctx) error) fiber.Router {
	return e.App.Get(path, handlers...)
}
func (e *Engine) Patch(path string, handlers ...func(*fiber.Ctx) error) fiber.Router {
	return e.App.Patch(path, handlers...)
}
func (e *Engine) Post(path string, handlers ...func(*fiber.Ctx) error) fiber.Router {
	return e.App.Post(path, handlers...)
}
func (e *Engine) Put(path string, handlers ...func(*fiber.Ctx) error) fiber.Router {
	return e.App.Put(path, handlers...)
}
func (e *Engine) Delete(path string, handlers ...func(*fiber.Ctx) error) fiber.Router {
	return e.App.Delete(path, handlers...)
}
func (e *Engine) Head(path string, handlers ...func(*fiber.Ctx) error) fiber.Router {
	return e.App.Head(path, handlers...)
}
func (e *Engine) Options(path string, handlers ...func(*fiber.Ctx) error) fiber.Router {
	return e.App.Options(path, handlers...)
}

func (e *Engine) Group(prefix string, handlers ...func(*fiber.Ctx) error) fiber.Router {
	return e.App.Group(prefix, handlers...)
}
