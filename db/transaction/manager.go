package transaction

import (
	"context"
	"errors"
)

var (
	ErrDBLookup = errors.New("matching database not found")
)

// NewManager 创建事务管理器.
//
// 能力:
//  1. Transaction 嵌套.
//  2. EscapeTransaction 事务逃逸.
//  3. OnCommitted 事务成功回调.
//
// 说明：
//
//	事务管理抽象实现, 业务代码需使用对应 DB Provider 提供的事务实现.
func NewManager(
	// 事务上下文在 context 中存储的 key.
	ctxKeyF func(context.Context) interface{},
	// 实现通过 context 查找 DB, 非事务上下文中 DB.
	lookupDB func(context.Context) interface{},
	// 实现事务执行并通过回调返回新 DB.
	transaction func(ctx context.Context, db interface{}, callback func(db interface{}) error) error,
) Manager {
	return &manager{
		ctxKeyF:     ctxKeyF,
		lookupDB:    lookupDB,
		transaction: transaction,
	}
}

type manager struct {
	// 返回在 context 中存储事务上下文的 Key.
	//
	// Key 需要有具体的类型，防止串信息.
	//
	// 例：
	// type tmkey string
	// func tmContextKey(ctx context.Context) interface{} {
	//    return tmkey("xxx")
	// }
	ctxKeyF func(context.Context) interface{}
	// 通过 context 查找 DB, 非事务上下文中 DB.
	lookupDB func(context.Context) interface{}
	// 实现事务开启并通过回调返回新 DB.
	transaction func(ctx context.Context, db interface{}, callback func(db interface{}) error) error
}

func (m *manager) findTransContext(ctx context.Context) *transContext {
	tc, ok := ctx.Value(m.ctxKeyF(ctx)).(*transContext)
	if !ok {
		return nil
	}
	return tc
}

func (m *manager) setTransContext(ctx context.Context, tc *transContext) context.Context {
	return context.WithValue(ctx, m.ctxKeyF(ctx), tc)
}

func (m *manager) cleanTransContext(ctx context.Context) context.Context {
	if m.findTransContext(ctx) == nil {
		return ctx
	}
	return context.WithValue(ctx, m.ctxKeyF(ctx), nil)
}

// findDBAndTransContext 查找 DB 和事务上下文.
func (m *manager) findDBAndTransContext(ctx context.Context) (*transContext, interface{}) {
	tc := m.findTransContext(ctx)
	if tc != nil {
		return tc, tc.db
	}
	db := m.lookupDB(ctx)
	return new(transContext), db
}

func (m *manager) Transaction(ctx context.Context, callback func(context.Context) error) error {
	var tc *transContext
	ptc, db := m.findDBAndTransContext(ctx)
	if db == nil {
		return ErrDBLookup
	}

	err := m.transaction(ctx, db, func(db interface{}) error {
		tc = ptc.Start(db)
		return callback(m.setTransContext(ctx, tc))
	})
	tc.End(err)
	return err
}

func (m *manager) EscapeTransaction(ctx context.Context, callback func(context.Context) error) error {
	return callback(m.cleanTransContext(ctx))
}

func (m *manager) OnCommitted(ctx context.Context, callback func(context.Context)) bool {
	tc := m.findTransContext(ctx)
	if tc == nil {
		// 未开启事务.
		return false
	}
	// 在事务外执行, 需要清理 context.
	tc.OnCommitted(func() { callback(m.cleanTransContext(ctx)) })
	return true
}
