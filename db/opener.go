package db

import (
	"context"
	"fmt"
	"net"
	"time"

	gosql "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const DialRetryTimes = 3

type MysqlDBOpener struct{}

func NewMysqlDBOpener() *MysqlDBOpener {
	return &MysqlDBOpener{}
}

func (m *MysqlDBOpener) DSN(opts *Options) string {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=PRC",
		opts.User, opts.Password, opts.Host, opts.Port, opts.Database,
	)
	if opts.Timeout > 0 {
		dsn = fmt.Sprintf("%s&timeout=%ds", dsn, opts.Timeout)
	}
	return dsn
}

func (m *MysqlDBOpener) Dialector(opts *Options) (gorm.Dialector, error) {
	dl := mysql.Open(m.DSN(opts))
	return dl, nil
}

func (m *MysqlDBOpener) OpenDB(opts *Options, rOpts *RuntimeOptions) (*gorm.DB, error) {
	// 注册建连函数，主要为了创建连接时可以进行重试
	gosql.RegisterDialContext("tcp", dialContextWithRetry)

	// 适配Logger接口
	conf := &gorm.Config{
		Logger: NewLoggerWrapper(
			rOpts.Logger,
			time.Duration(opts.SlowThreshold)*time.Millisecond,
			logger.LogLevel(opts.LogLevel),
		),
		QueryFields:              true,
		DisableNestedTransaction: true,
	}

	// 开启DB对象
	db, err := gorm.Open(mysql.Open(m.DSN(opts)), conf)
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(opts.MaxOpen)
	sqlDB.SetMaxIdleConns(opts.MaxIdle)
	sqlDB.SetConnMaxLifetime(time.Duration(opts.Lifetime) * time.Minute)

	// 注册插件
	err = m.registerPlugins(db, opts, rOpts)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func (m *MysqlDBOpener) registerPlugins(db *gorm.DB, opts *Options, rOpts *RuntimeOptions) error {
	// 用户自定义插件
	dbPlugins := rOpts.Plugins

	// 注册插件
	for _, plugin := range dbPlugins {
		if err := db.Use(plugin); err != nil {
			return err
		}
	}
	return nil
}

// 重载建连函数，在原来的基础上增加重试，最多重试三次
// 目前重试次数 & Logger 没有开放配置，主要原因是目前只开放了注册全局函数，基于现有条件区分不开
func dialContextWithRetry(ctx context.Context, addr string) (conn net.Conn, err error) {
	for i := 0; i < DialRetryTimes; i++ {
		if i > 0 {
			time.Sleep(time.Millisecond * 50)
		}
		nd := net.Dialer{}
		conn, err = nd.DialContext(ctx, "tcp", addr)
		if err == nil {
			break
		}

	}
	return
}
