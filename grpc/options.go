package grpc

import (
	"sync"
	"time"

	auth "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type Options struct {
	lock               sync.RWMutex
	Grpc               *Grpc                         `id:"grpc" json:"grpc"`
	Parameters         *Parameters                   `id:"parameters" json:"parameters"`
	Auth               *Auth                         `id:"-"`
	CustomInterceptors []grpc.UnaryServerInterceptor `id:"-" json:"-"`
}

type Grpc struct {
	Network string `id:"network" json:"network" default:"tcp"`
	Addr    string `id:"addr" json:"addr" default:":50051"`
}

type Parameters struct {
	IdleTimeout       time.Duration `id:"idle_timeout" json:"idle_timeout" desc:"Idle timeout for connections"`
	MaxLifeTime       time.Duration `id:"max_life_time" json:"max_life_time" desc:"Max life time for connections"`
	ForceCloseWait    time.Duration `id:"force_close_wait" json:"force_close_wait" desc:"Time to wait before force closing connections"`
	KeepAliveInterval time.Duration `id:"keep_alive_interval" json:"keep_alive_interval" desc:"Interval to send keep alive messages"`
	KeepAliveTimeout  time.Duration `id:"keep_alive_timeout" json:"keep_alive_timeout" desc:"Timeout for keep alive messages"`
}

type Auth struct {
	Options auth.AuthFunc
}

func DefaultOptions() *Options {
	return &Options{
		Grpc:       DefaultGrpcOptions(),
		Parameters: DefaultParametersOptions(),
	}
}

func DefaultGrpcOptions() *Grpc {
	return &Grpc{
		Network: "tcp",
		Addr:    ":50051",
	}
}

func DefaultParametersOptions() *Parameters {
	return &Parameters{
		IdleTimeout:       time.Second * 60,
		MaxLifeTime:       time.Hour * 2,
		ForceCloseWait:    time.Second * 20,
		KeepAliveInterval: time.Second * 60,
		KeepAliveTimeout:  time.Second * 20,
	}
}

func (o *Options) merge(options []*Options) {
	o.lock.Lock()
	defer o.lock.Unlock()

	if len(options) == 0 {
		return
	}

	for _, v := range options {
		if v == nil {
			continue
		}

		if v.Grpc != nil {
			o.Grpc = v.Grpc
		}

		if v.Parameters != nil {
			o.Parameters = v.Parameters
		}
		if len(v.CustomInterceptors) > 0 {
			o.CustomInterceptors = append(o.CustomInterceptors, v.CustomInterceptors...)
		}
	}
}

func (o *Parameters) getGrpcKeepaliveParams() grpc.ServerOption {
	return grpc.KeepaliveParams(
		keepalive.ServerParameters{
			MaxConnectionIdle:     o.IdleTimeout,
			MaxConnectionAgeGrace: o.ForceCloseWait,
			Time:                  o.KeepAliveInterval,
			Timeout:               o.KeepAliveTimeout,
			MaxConnectionAge:      o.MaxLifeTime,
		},
	)
}
