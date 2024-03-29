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
	"log"
)

type NoOpReaderWriter[T Cacheable] struct {
	getTemplateItem func(key interface{}) T
	panicOnLoad     bool // for testing error handling
	panicOnWrite    bool // for testing error handling
}

// Check interface is complete
var _ CacheReaderWriter[string, EmptyCacheable] = (*NoOpReaderWriter[EmptyCacheable])(nil)

func NewNoOpReaderWriter[T Cacheable](itemTemplate func(key any) T, forcePanics ...bool) NoOpReaderWriter[T] {
	doPanics := len(forcePanics) > 0 && forcePanics[0]
	return NoOpReaderWriter[T]{
		getTemplateItem: itemTemplate,
		panicOnWrite:    doPanics,
		panicOnLoad:     doPanics,
	}
}

func (g NoOpReaderWriter[T]) Find(key string, _ interface{}) (T, error) {
	template := g.getTemplateItem(key)
	if g.panicOnLoad {
		panic("test panic, read")
	}
	return template, errors.New("NoOp, item not found")
}

func (g NoOpReaderWriter[T]) Save(_ T, _ interface{}) error {
	if g.panicOnWrite {
		panic("test panic, write")
	}
	return nil
}

func (g NoOpReaderWriter[T]) BeginTx() (tx interface{}, err error) {
	tx = "transaction"
	return tx, nil
}

func (g NoOpReaderWriter[T]) CommitTx(_ interface{}) {
	return
}

func (g NoOpReaderWriter[T]) Info(msg string, _ string, _ ...T) {
	log.Print("[info] ", msg)
}

func (g NoOpReaderWriter[T]) Warn(msg string, _ string, _ ...T) {
	log.Print("[warn] ", msg)
}
