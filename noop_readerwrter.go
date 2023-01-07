/*
 * Copyright (c) 2023. Vade Mecum Ltd.  All Rights Reserved.
 */

package lazywritercache

import (
	"errors"
	"log"
)

type NoOpReaderWriter struct {
	getTemplateItem func(key interface{}) CacheItem
	panicOnLoad     bool // for testing error handling
	panicOnWrite    bool // for testing error handling
}

func NewNoOpReaderWriter(itemTemplate func(key interface{}) CacheItem, forcePanics ...bool) NoOpReaderWriter {
	doPanics := len(forcePanics) > 0 && forcePanics[0]
	return NoOpReaderWriter{
		getTemplateItem: itemTemplate,
		panicOnWrite:    doPanics,
		panicOnLoad:     doPanics,
	}
}

func (g NoOpReaderWriter) Find(key interface{}, _ interface{}) (CacheItem, error) {
	template := g.getTemplateItem(key)
	if g.panicOnLoad {
		panic("test panic, read")
	}
	return template, errors.New("NoOp, item not found")
}

func (g NoOpReaderWriter) Save(_ CacheItem, _ interface{}) error {
	if g.panicOnWrite {
		panic("test panic, write")
	}
	return nil
}

func (g NoOpReaderWriter) BeginTx() (tx interface{}, err error) {
	tx = "transaction"
	return tx, nil
}

func (g NoOpReaderWriter) CommitTx(_ interface{}) {
	return
}

func (g NoOpReaderWriter) Info(msg string) {
	log.Print("[info] ", msg)
}

func (g NoOpReaderWriter) Warn(msg string) {
	log.Print("[warn] ", msg)
}
