package transaction

import (
	"context"
	"sync"
)

// Manager 定义事务管理器.
//
// 此事务管理器具有跨层事务能力.
//
// 具体实现由资源提供方提供，如: db.Provider 的具体实现.
type Manager interface {
	// Transaction 事务内执行回调.
	//
	// 事务在 context 进行标记.
	// 回调执行 panic 时，事务正确回滚.
	//
	// Transaction 可嵌套使用, Transaction 实现为 SavePoint.
	//
	// 回调 context 不要在新 goroutine 或回调范围外使用.
	//
	// 新的 goroutinue 或 callback 外使用回调中的 context，使用 EscapeTransaction
	// 清除标记.
	Transaction(ctx context.Context, callback func(context.Context) error) error

	// EscapeTransaction 使回调逃脱当前事务.
	//
	// 回调 context 事务标记已被清除.
	//
	// 逃脱当前事务后, OnCommitted 注册失效. 需要开启新事务才可注册.
	EscapeTransaction(ctx context.Context, callback func(context.Context) error) error

	// OnCommitted 事务提交成功后回调.
	//
	// 注册成功返回 true, 注册失败返回 false.
	//
	// 当前事务及其上级事务都成功时回调.
	//
	// OnCommitted 需在 Transaction callback 中使用回调的 context 进行注册.
	OnCommitted(ctx context.Context, callback func(context.Context)) bool
}

// TransContext 代表事务上下文.
//
// 用于事务管理器的具体实现从上下文中获取事务 DB.
type TransContext interface {
	// 获取事务 DB.
	GetTransDB() interface{}
}

// transContext 实现事务上下文.
type transContext struct {
	// 根节点属性.
	mut                  sync.Mutex
	onCommittedCallbacks []func()

	// 父节点.
	//
	// 父节点为 nil，则为根节点.
	parent *transContext
	// 当前事务 DB 实例.
	db interface{}
	// 是否 panic.
	//
	// 事务开始前设置为 true , 事务结束时设置为 false.
	paniced bool
	// 当前事务执行结果是否异常.
	err error
}

//var _ TransContext = new(transContext)

// 获取事务 DB.
func (tc *transContext) GetTransDB() interface{} {
	return tc.db
}

// Start 标记新事务开启.
func (tc *transContext) Start(db interface{}) *transContext {
	return &transContext{parent: tc, db: db, paniced: true}
}

// End 标记当前事务结束.
func (tc *transContext) End(err error) {
	if tc == nil {
		return
	}
	tc.paniced = false
	tc.err = err
	tc.doOnCommittedCallbacks()
}

// OnCommitted 添加事务回调. 注册至根节点当中
func (tc *transContext) OnCommitted(cb func()) {
	callback := func() {
		if tc.isCommitted() {
			cb()
		}
	}
	if tc.parent == nil {
		tc.mut.Lock()
		defer tc.mut.Unlock()

		tc.onCommittedCallbacks = append(tc.onCommittedCallbacks, callback)
		return
	}
	tc.parent.OnCommitted(callback)
}

// isRoot 返回是否根事务节点.
func (tc *transContext) isRoot() bool {
	return tc.parent == nil
}

// isCommitted 判断当前节点事务是否成功提交.
func (tc *transContext) isCommitted() bool {
	if tc.paniced {
		return false
	}
	if tc.err != nil {
		return false
	}
	return true
}

// doOnCommittedCallbacks 处理注册到根节点的回调.
func (tc *transContext) doOnCommittedCallbacks() {
	// 非根事务节点不触发.
	if !tc.isRoot() {
		return
	}

	var callbacks []func()
	tc.mut.Lock()
	for _, callback := range tc.onCommittedCallbacks {
		callbacks = append(callbacks, callback)
	}
	tc.mut.Unlock()

	for _, callback := range callbacks {
		callback()
	}
}
