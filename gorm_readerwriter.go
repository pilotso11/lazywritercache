// MIT License
//
// Copyright (c) 2023 Seth Osher
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package lazywritercache

import (
	"errors"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// GormCacheReaderWriter is the GORM implementation of the CacheReaderWriter.   It should work with any DB GORM supports.
// It's been tested with Postgres and Mysql.   UseTransactions should be set to true unless you have a really good reason not to.
// If set to true t find and save operation is done in a single transaction which ensures no collisions with a parallel writer.
// But also the flush is done in a transaction which is much faster.  You don't really want to set this to false except for debugging.
type GormCacheReaderWriter[K comparable, T Cacheable] struct {
	db                *gorm.DB
	finderWhereClause string
	getTemplateItem   func(key K) T
	logger            *zap.Logger
	UseTransactions   bool
}

// Check interface is complete
var _ CacheReaderWriter[string, EmptyCacheable] = (*GormCacheReaderWriter[string, EmptyCacheable])(nil)

// NewGormCacheReaderWriter creates a GORM Cache Reader Writer supply a new item creator and a wrapper to db.Save() that first unwraps item Cacheable to your type
func NewGormCacheReaderWriter[K comparable, T Cacheable](db *gorm.DB, logger *zap.Logger, keyField string,
	itemTemplate func(key K) T) GormCacheReaderWriter[K, T] {
	return GormCacheReaderWriter[K, T]{
		db:                db,
		logger:            logger,
		finderWhereClause: keyField + " = ?",
		getTemplateItem:   itemTemplate,
		UseTransactions:   true,
	}
}

func (g GormCacheReaderWriter[K, T]) Find(key K, tx any) (T, error) {
	var dbTx *gorm.DB
	if tx == nil {
		dbTx = g.db
	} else {
		dbTx = tx.(*gorm.DB)
	}

	template := g.getTemplateItem(key)

	res := dbTx.Limit(1).Find(&template, g.finderWhereClause, key)

	if res.Error != nil {
		return template, res.Error
	}
	if res.RowsAffected == 0 {
		return template, errors.New("not found")
	}
	return template, nil
}

func (g GormCacheReaderWriter[K, T]) Save(item T, tx any) error {
	dbTx := tx.(*gorm.DB)
	// Generics to the rescue!?
	res := dbTx.Save(&item)
	return res.Error
}

func (g GormCacheReaderWriter[K, T]) BeginTx() (tx any, err error) {
	if g.UseTransactions {
		tx = g.db.Begin()
		return tx, nil
	}
	return g.db, nil
}

func (g GormCacheReaderWriter[K, T]) CommitTx(tx any) {
	dbTx := tx.(*gorm.DB)
	if g.UseTransactions {
		dbTx.Commit()
	}
	return
}

func (g GormCacheReaderWriter[K, T]) Info(msg string) {
	g.logger.Info(msg)
}

func (g GormCacheReaderWriter[K, T]) Warn(msg string) {
	g.logger.Warn(msg)
}
