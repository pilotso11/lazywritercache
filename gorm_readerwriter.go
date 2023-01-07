/*
 * Copyright (c) 2023. Vade Mecum Ltd.  All Rights Reserved.
 */

package lazywritercache

import (
	"errors"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type GormCacheReaderWriter struct {
	db                *gorm.DB
	finderWhereClause string
	getTemplateItem   func(key interface{}) CacheItem
	logger            *zap.Logger
	useTransactions   bool
	saver             func(tx *gorm.DB, item CacheItem) (result *gorm.DB)
	finder            func(tx *gorm.DB, item CacheItem) *gorm.DB
}

type CacheItemConstraint interface {
	CacheItem
}

func finder[T CacheItemConstraint](tx *gorm.DB, item T, queryString string) (result *gorm.DB) {
	return tx.Limit(1).Find(&item, queryString, item.Key())
}
func saver[T CacheItemConstraint](tx *gorm.DB, item T) *gorm.DB {
	return tx.Save(&item)
}

// NewGormCacheReaderWriter creates a GORM Cache Reader Writer supply a new item creator and a wrapper to db.Save() that first unwraps item CacheItem to your type
func NewGormCacheReaderWriter(db *gorm.DB, logger *zap.Logger, keyField string,
	itemTemplate func(key interface{}) CacheItem,
	saver func(tx *gorm.DB, item CacheItem) (result *gorm.DB),
	finder func(tx *gorm.DB, item CacheItem) *gorm.DB) GormCacheReaderWriter {
	return GormCacheReaderWriter{
		db:                db,
		logger:            logger,
		finderWhereClause: keyField + " = ?",
		getTemplateItem:   itemTemplate,
		useTransactions:   false,
		saver:             saver,
		finder:            finder,
	}
}

func (g GormCacheReaderWriter) Find(key interface{}, tx interface{}) (CacheItem, error) {
	var dbTx *gorm.DB
	if tx == nil {
		dbTx = g.db
	} else {
		dbTx = tx.(*gorm.DB)
	}

	template := g.getTemplateItem(key)

	res := g.finder(dbTx, template)

	if res.Error != nil {
		return template, res.Error
	}
	if res.RowsAffected == 0 {
		return template, errors.New("not found")
	}
	return template, nil
}

func (g GormCacheReaderWriter) Save(item CacheItem, tx interface{}) error {
	dbTx := tx.(*gorm.DB)
	// Sadly I can't just call db.Save(&item) here as it's the wrong object type
	// dbTx.Save(&item)
	res := g.saver(dbTx, item)
	return res.Error
}

func (g GormCacheReaderWriter) BeginTx() (tx interface{}, err error) {
	if g.useTransactions {
		tx = g.db.Begin()
		return tx, nil
	}
	return g.db, nil
}

func (g GormCacheReaderWriter) CommitTx(tx interface{}) {
	dbTx := tx.(*gorm.DB)
	if g.useTransactions {
		dbTx.Commit()
	}
	return
}

func (g GormCacheReaderWriter) Info(msg string) {
	g.logger.Info(msg)
}

func (g GormCacheReaderWriter) Warn(msg string) {
	g.logger.Warn(msg)
}
