package db

import (
	"context"
	"math/rand"
	"strconv"

	"github.com/tp-life/driver/db/transaction"

	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

// Provider 定义 *gorm.DB 提供者.
//
// 例：
//
//	特定租户访问单独的数据库集群,其他租户访问默认集群.
type Provider interface {
	// UseDB 实现通过 context 选择数据库.
	// 如果在事务上下文内，返回写库.
	// 不在事务上下文内时, 依据执行语句动态选择读库或写库.
	// 无匹配 DB 时返回 nil.
	UseDB(context.Context) *gorm.DB

	// UseWriteDB 实现通过 context 选择写库.
	// 无匹配 DB 时返回 nil.
	UseWriteDB(context.Context) *gorm.DB
}

// SourceBuilder 创建数据源的构造器
type SourceBuilder interface {
	ToSource(rOpts *RuntimeOptions) (Source, error)
}

// NewProvider 创建支持事务管理的 db.Provider
// Scopes 在新会话创建后通过 db.Scopes(Scopes...) 应用.
// 数据源、Provider、事务管理、插件集成参照:
// Notice: 不同 Provider 间的事务不共享.
func NewProvider(opts SourceBuilder, rOptsList ...*RuntimeOptions) *TransProvider {
	var rOpts *RuntimeOptions
	if len(rOptsList) > 0 {
		rOpts = rOptsList[0]
	}

	if rOpts == nil {
		rOpts = &RuntimeOptions{}
	}
	if len(rOpts.Plugins) == 0 {
		rOpts.Plugins = []gorm.Plugin{}
	}

	src, err := opts.ToSource(rOpts)
	if err != nil {
		panic(err)
	}

	p := &TransProvider{
		Source:   src,
		scopes:   rOpts.Scopes,
		txSuffix: strconv.FormatInt(rand.Int63(), 10),
	}
	lookupDB := func(ctx context.Context) interface{} {
		return p.lookupDB(ctx, true)
	}
	p.Manager = transaction.NewManager(p.getCtxKey, lookupDB, p.transaction)
	return p
}

// ToProvider 转换 *TransProvider 为 Provider.
// 用于依赖注入的工厂函数.
func ToProvider(tp *TransProvider) Provider {
	return tp
}

// ToTransactionManager 转换 *TransProvider 为 transaction.Manager.
// 用于依赖注入的工厂函数.
func ToTransactionManager(tp *TransProvider) transaction.Manager {
	return tp
}

// TransProvider 实现支持事务上下文的 DB Provider.
type TransProvider struct {
	Source
	transaction.Manager

	txSuffix string
	scopes   []func(*gorm.DB) *gorm.DB
}

//var _ transaction.Manager = new(TransProvider)

type transCtxKey string

// getCtxKey 返回事务上下文存储到 context 的 Key.
// 事务上下文实现了 transaction.TransContext
// 返回的 key 需要转换为私有类型, 防止内容污染.
func (p *TransProvider) getCtxKey(ctx context.Context) interface{} {
	name := p.getWriteDBName(ctx)
	return transCtxKey(name + "." + p.txSuffix)
}

// lookupDB 查找非事务上下文 DB.
func (p *TransProvider) lookupDB(ctx context.Context, write bool) *gorm.DB {
	if write {
		return p.getWriteDB(ctx).Clauses(dbresolver.Write)
	}
	return p.getReadDB(ctx)
}

// findTransDB 查找事务上下文 DB.
func (p *TransProvider) findTransDB(ctx context.Context) *gorm.DB {
	tc, ok := ctx.Value(p.getCtxKey(ctx)).(transaction.TransContext)
	if !ok {
		return nil
	}
	return tc.GetTransDB().(*gorm.DB)
}

// isInTransaction 判断当前 context 是否在事务上下文.
func (p *TransProvider) isInTransaction(ctx context.Context) bool {
	return p.findTransDB(ctx) != nil
}

// transaction 执行数据库事务.
func (p *TransProvider) transaction(ctx context.Context, db interface{}, callback func(db interface{}) error) error {
	if p.isInTransaction(ctx) {
		return callback(db)
	}
	return db.(*gorm.DB).Transaction(func(db *gorm.DB) error {
		return callback(db)
	})
}

func (p *TransProvider) useDB(ctx context.Context, write bool) *gorm.DB {
	db := p.findTransDB(ctx)
	if db == nil {
		db = p.lookupDB(ctx, write)
	}
	if db == nil {
		return nil
	}
	// 默认不支持嵌套事务
	sess := &gorm.Session{Context: ctx, DisableNestedTransaction: true}

	return db.Session(sess).Scopes(p.scopes...)
}

// UseDB 实现通过 context 选择数据库.
// 如果在事务上下文内，返回写库.
// 不在事务上下文内时, 依据执行语句动态选择读库或写库.
// 无匹配 DB 时 panic.
func (p *TransProvider) UseDB(ctx context.Context) *gorm.DB {
	return p.useDB(ctx, false)
}

// UseWriteDB 实现通过 context 选择写库.
// 无匹配 DB 时 panic.
func (p *TransProvider) UseWriteDB(ctx context.Context) *gorm.DB {
	return p.useDB(ctx, true)
}
