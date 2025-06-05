// MIT License
//
// Copyright (c) LF0LF3 Seth Osher
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

package lazygormcache

import (
	"errors"
	"log"

	"gorm.io/gorm"

	"github.com/pilotso11/lazywritercache/lockfree"
)

// LoggerLF is an interface that can be implemented to provide logging for the cache.
// The default logger is log.Println.
type LoggerLF interface {
	Info(msg string, action string, item ...lockfree.CacheableLF)
	Warn(msg string, action string, item ...lockfree.CacheableLF)
	Error(msg string, action string, item ...lockfree.CacheableLF)
}

// ReaderWriteLF is the GORM implementation of the CacheReaderWriter.   It should work with any DB GORM supports.
// It's been tested with Postgres and Mysql.   UseTransactions should be set to true unless you have a really good reason not to.
// If set to true t find and save operation is done in a single transaction which ensures no collisions with a parallel writer.
// But also the flush is done in a transaction which is much faster.  You don't really want to set this to false except for debugging.
type ReaderWriteLF[T lockfree.CacheableLF] struct {
	db              *gorm.DB
	getTemplateItem func(key string) T
	UseTransactions bool
	Logger          LoggerLF
}

// Check interface is complete
var _ lockfree.CacheReaderWriterLF[lockfree.EmptyCacheableLF] = (*ReaderWriteLF[lockfree.EmptyCacheableLF])(nil)

// NewGormCacheReaderWriteLF creates a GORM Cache Reader Writer supply a new item creator and a wrapper to db.Save() that first unwraps item CacheableLF to your type
func NewGormCacheReaderWriteLF[T lockfree.CacheableLF](db *gorm.DB, itemTemplate func(key string) T) ReaderWriteLF[T] {
	return ReaderWriteLF[T]{
		db:              db,
		getTemplateItem: itemTemplate,
		UseTransactions: true,
	}
}

func (g ReaderWriteLF[T]) Find(key string, tx any) (T, error) {
	var dbTx *gorm.DB
	if tx == nil {
		dbTx = g.db
	} else {
		dbTx = tx.(*gorm.DB)
	}

	template := g.getTemplateItem(key)

	res := dbTx.Limit(1).Find(&template, &template)

	if res.Error != nil {
		return template, res.Error
	}
	if res.RowsAffected == 0 {
		return template, errors.New("not found")
	}
	return template, nil
}

func (g ReaderWriteLF[T]) Save(item T, tx any) error {
	dbTx := tx.(*gorm.DB)
	// Generics to the rescue!?
	res := dbTx.Save(&item)
	return res.Error
}

func (g ReaderWriteLF[T]) BeginTx() (tx any, err error) {
	if g.UseTransactions {
		tx = g.db.Begin()
		return tx, nil
	}
	return g.db, nil
}

func (g ReaderWriteLF[T]) CommitTx(tx any) {
	dbTx := tx.(*gorm.DB)
	if g.UseTransactions {
		dbTx.Commit()
	}
	return
}

func (g ReaderWriteLF[T]) Info(msg string, action string, item ...T) {
	if g.Logger != nil {
		if len(item) > 0 {
			g.Logger.Info(msg, action, item[0])
		} else {
			g.Logger.Info(msg, action)
		}
	} else {
		log.Println("[info] ", msg)
	}
}

func (g ReaderWriteLF[T]) Warn(msg string, action string, item ...T) {
	if g.Logger != nil {
		if len(item) > 0 {
			g.Logger.Warn(msg, action, item[0])
		} else {
			g.Logger.Warn(msg, action)
		}
	} else {
		log.Println("[warn] ", msg)
	}
}
