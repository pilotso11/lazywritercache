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
	"context"
	"errors"
	"log"
	"strings"

	"github.com/puzpuzpuz/xsync"
	"gorm.io/gorm"

	"github.com/pilotso11/lazywritercache"
)

// LoggerLF is an interface that can be implemented to provide logging for the cache.
// The default logger is log.Println.
type LoggerLF[K comparable] interface {
	Info(ctx context.Context, msg string, action lazywritercache.CacheAction, item ...lazywritercache.CacheableLF[K])
	Warn(ctx context.Context, msg string, action lazywritercache.CacheAction, item ...lazywritercache.CacheableLF[K])
	Error(ctx context.Context, msg string, action lazywritercache.CacheAction, item ...lazywritercache.CacheableLF[K])
}

// ReaderWriteLF is the GORM implementation of the CacheReaderWriter.   It should work with any DB GORM supports.
// It's been tested with Postgres and Mysql.   UseTransactions should be set to true unless you have a really good reason not to.
// If set to true t find and save operation is done in a single transaction which ensures no collisions with a parallel writer.
// But also the flush is done in a transaction which is much faster.  You don't really want to set this to false except for debugging.
type ReaderWriteLF[K comparable, T lazywritercache.CacheableLF[K]] struct {
	db                *gorm.DB
	getTemplateItem   func(key K) T
	UseTransactions   bool
	Logger            LoggerLF[K]
	RecoverableErrors *xsync.MapOf[string, error]
}

// Check interface is complete
var _ lazywritercache.CacheReaderWriterLF[string, lazywritercache.EmptyCacheableLF] = (*ReaderWriteLF[string, lazywritercache.EmptyCacheableLF])(nil)

// NewReaderWriterLF creates a GORM Cache Reader Writer supply a new item creator and a wrapper to db.Save() that first unwraps item CacheableLF to your type
func NewReaderWriterLF[K comparable, T lazywritercache.CacheableLF[K]](db *gorm.DB, itemTemplate func(key K) T) ReaderWriteLF[K, T] {
	return ReaderWriteLF[K, T]{
		db:                db,
		getTemplateItem:   itemTemplate,
		UseTransactions:   true,
		RecoverableErrors: xsync.NewMapOf[error](),
	}
}

func (g ReaderWriteLF[K, T]) Find(ctx context.Context, key K, tx any) (T, error) {
	var dbTx *gorm.DB
	if tx == nil {
		dbTx = g.db
	} else {
		dbTx = tx.(*gorm.DB)
	}

	template := g.getTemplateItem(key)

	res := dbTx.Limit(1).WithContext(ctx).Find(&template, &template)

	if res.Error != nil {
		return template, res.Error
	}
	if res.RowsAffected == 0 {
		return template, errors.New("not found")
	}
	return template, nil
}

func (g ReaderWriteLF[K, T]) Save(ctx context.Context, item T, tx any) error {
	dbTx := tx.(*gorm.DB)
	// Generics to the rescue!?
	res := dbTx.WithContext(ctx).Save(&item)
	return res.Error
}

func (g ReaderWriteLF[K, T]) BeginTx(ctx context.Context) (tx any, err error) {
	if g.UseTransactions {
		tx = g.db.WithContext(ctx).Begin()
		return tx, nil
	}
	return g.db, nil
}

func (g ReaderWriteLF[K, T]) CommitTx(ctx context.Context, tx any) (err error) {
	dbTx := tx.(*gorm.DB)
	if g.UseTransactions {
		err = dbTx.WithContext(ctx).Commit().Error
	}
	return err
}

func (g ReaderWriteLF[K, T]) RollbackTx(ctx context.Context, tx any) (err error) {
	dbTx := tx.(*gorm.DB)
	if g.UseTransactions {
		err = dbTx.WithContext(ctx).Rollback().Error
	}
	return err
}

func (g ReaderWriteLF[K, T]) Info(ctx context.Context, msg string, action lazywritercache.CacheAction, item ...T) {
	if g.Logger != nil {
		if len(item) > 0 {
			g.Logger.Info(ctx, msg, action, item[0])
		} else {
			g.Logger.Info(ctx, msg, action)
		}
	} else {
		log.Println("[info] ", msg)
	}
}

func (g ReaderWriteLF[K, T]) Warn(ctx context.Context, msg string, action lazywritercache.CacheAction, item ...T) {
	if g.Logger != nil {
		if len(item) > 0 {
			g.Logger.Warn(ctx, msg, action, item[0])
		} else {
			g.Logger.Warn(ctx, msg, action)
		}
	} else {
		log.Println("[warn] ", msg)
	}
}

func (g ReaderWriteLF[K, T]) IsRecoverable(_ context.Context, err error) bool {
	switch {
	case errors.Is(err, gorm.ErrInvalidData):
		return false
	case errors.Is(err, gorm.ErrDuplicatedKey):
		return false
	case errors.Is(err, gorm.ErrCheckConstraintViolated):
		return false
	case errors.Is(err, gorm.ErrForeignKeyViolated):
		return false
	case errors.Is(err, gorm.ErrInvalidTransaction):
		return false
	case errors.Is(err, gorm.ErrPrimaryKeyRequired):
		return false
	case strings.Contains(strings.ToLower(err.Error()), "deadlock"):
		return true
	default:
		return false
	}
}

func (g ReaderWriteLF[K, T]) Fail(_ context.Context, _ error, _ ...T) {
	// nothing we can do here.
}
