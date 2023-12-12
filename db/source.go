package db

import (
	"context"

	"gorm.io/gorm"
)

// Source 代表数据源.
type Source interface {
	// 获取写库名.
	getWriteDBName(context.Context) string
	// 获取写库.
	getWriteDB(context.Context) *gorm.DB
	// 获取读库名.
	getReadDBName(context.Context) string
	// 获取读库.
	getReadDB(context.Context) *gorm.DB
}

// source 代表数据源.
type source struct {
	writeDBName string
	writeDB     *gorm.DB
	readDBName  string
	readDB      *gorm.DB
}

// NewSource 创建单库数据源.
func NewSource(name string, db *gorm.DB) Source {
	return &source{writeDBName: name, writeDB: db, readDBName: name, readDB: db}
}

// NewWriteReadSource 创建读写分离数据源.
func NewWriteReadSource(writeDBName string, writeDB *gorm.DB, readDBName string, readDB *gorm.DB) Source {
	return &source{writeDBName: writeDBName, writeDB: writeDB, readDBName: readDBName, readDB: readDB}
}

// 获取写库名.
func (s *source) getWriteDBName(ctx context.Context) string {
	return s.writeDBName
}

// 获取写库.
func (s *source) getWriteDB(ctx context.Context) *gorm.DB {
	return s.writeDB
}

// 获取读库名.
func (s *source) getReadDBName(ctx context.Context) string {
	return s.readDBName
}

// 获取读库.
func (s *source) getReadDB(ctx context.Context) *gorm.DB {
	return s.readDB
}
