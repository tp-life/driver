package db

import (
	"errors"
	"fmt"
	"log/slog"

	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

var (
	ErrWriteDBNotConfigured = errors.New("write database not configured")
)

// DBOpener 数据库连接创建
type DBOpener interface {
	Dialector(opts *Options) (gorm.Dialector, error)
	OpenDB(opts *Options, rOpts *RuntimeOptions) (*gorm.DB, error)
}

// RWOptions 定义主从配置.
type RWOptions struct {
	// 主库配置.
	Write *Options `json:"write"`
	// 从库配置.
	Read *Options `json:"read"`
}

// Options 定义数据库配置.
type Options struct {
	Host string `id:"mysql_host" json:"mysql_host"`
	Port int    `id:"mysql_port" json:"mysql_port"`

	Database string `id:"mysql_database" json:"mysql_database"`
	User     string `id:"mysql_user" json:"mysql_user"`
	Password string `id:"mysql_password" json:"mysql_password"`
	Timeout  int    `id:"mysql_conn_timeout" json:"mysql_conn_timeout" default:"3"` // 单位：秒

	MaxOpen  int `id:"mysql_max_open" json:"mysql_max_open" default:"128"`
	MaxIdle  int `id:"mysql_max_idle" json:"mysql_max_idle" default:"8"`
	Lifetime int `id:"mysql_conn_livetime" json:"mysql_conn_livetime" default:"60"` // 单位：分钟

	Retry         int  `id:"mysql_retry" json:"mysql_retry" default:"3"`         // deprecated 原来初始化连接时的重试次数，已废弃
	Tracing       bool `id:"mysql_tracing" json:"mysql_tracing" default:"false"` // 是否开启链路追踪
	LogLevel      int  `id:"log_level" json:"log_level" default:"3"`             // 日志级别，默认为warning
	SlowThreshold int  `id:"slow_threshold" json:"slow_threshold" default:"500"` // 慢查询阈值，单位：毫秒
}

// RuntimeOptions 不从配置文件加载，通常需要在代码中初始化的配置
type RuntimeOptions struct {
	Logger  slog.Logger
	Plugins []gorm.Plugin             // gorm 插件，默认会有 Logger -> Metrics，不需要额外传
	Scopes  []func(*gorm.DB) *gorm.DB // 全局 scope 函数
}

// OpenDB 创建数据库连接.
func (o *RWOptions) openDB(opener DBOpener, rOpts *RuntimeOptions) (*gorm.DB, error) {
	if o.Write == nil {
		return nil, ErrWriteDBNotConfigured
	}
	db, err := opener.OpenDB(o.Write, rOpts)
	if err != nil {
		return nil, err
	}

	if o.Read == nil {
		return db, nil
	}
	rd, err := opener.Dialector(o.Read)
	if err != nil {
		return nil, err
	}

	if err := db.Use(
		dbresolver.Register(
			dbresolver.Config{
				Replicas: []gorm.Dialector{rd},
			},
		),
	); err != nil {
		return nil, err
	}
	return db, nil
}

// ToSource 转换配置为数据源.
func (o *RWOptions) ToSource(rOpts *RuntimeOptions) (Source, error) {
	return o.toSource(NewMysqlDBOpener(), rOpts) // opener 暂不开放扩展
}

func (o *RWOptions) toSource(opener DBOpener, rOpts *RuntimeOptions) (Source, error) {
	db, err := o.openDB(opener, rOpts)
	if err != nil {
		return nil, err
	}
	name := o.Write.fullName()
	return NewSource(name, db), nil
}

// ToSource 转换配置为数据源.
func (o *Options) ToSource(rOpts *RuntimeOptions) (Source, error) {
	return o.toSource(NewMysqlDBOpener(), rOpts) // opener 暂不开放扩展
}

func (o *Options) toSource(opener DBOpener, rOpts *RuntimeOptions) (Source, error) {
	db, err := opener.OpenDB(o, rOpts)
	if err != nil {
		return nil, err
	}
	return NewSource(o.fullName(), db), nil
}

func (o *Options) fullName() string {
	if o == nil {
		return ""
	}
	return fmt.Sprintf("%s:%d/%s", o.Host, o.Port, o.Database)
}
