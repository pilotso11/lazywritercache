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

package lazywritercache

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync/atomic"
)

// NoOpReaderWriterLF is a mock ReaderWriter for the lock free version of lazy writer cache.
type NoOpReaderWriterLF[K comparable, T CacheableLF[K]] struct {
	getTemplateItem func(key K) T
	panicOnNext     *atomic.Bool
	errorOnNext     *atomic.Value
	warnCount       *atomic.Int64
	infoCount       *atomic.Int64
	fails           *atomic.Int64
	logBuffer       *strings.Builder
}

// Check interface is complete
var _ CacheReaderWriterLF[string, EmptyCacheableLF] = (*NoOpReaderWriterLF[string, EmptyCacheableLF])(nil)

func NewNoOpReaderWriterLF[K comparable, T CacheableLF[K]](itemTemplate func(key K) T) NoOpReaderWriterLF[K, T] {
	var buf strings.Builder
	log.SetOutput(&buf)
	return NoOpReaderWriterLF[K, T]{
		getTemplateItem: itemTemplate,
		panicOnNext:     &atomic.Bool{},
		errorOnNext:     &atomic.Value{},
		warnCount:       &atomic.Int64{},
		infoCount:       &atomic.Int64{},
		fails:           &atomic.Int64{},
		logBuffer:       &buf,
	}
}

func (g NoOpReaderWriterLF[K, T]) Find(_ context.Context, key K, _ any) (T, error) {
	if g.panicOnNext.CompareAndSwap(true, false) {
		panic("test panic, write")
	}
	msg := g.errorOnNext.Load()
	if msg != nil && strings.Contains(msg.(string), "find") {
		g.removeFromErrorOnNext()
		return g.getTemplateItem(key), errors.New("write " + msg.(string))
	}
	template := g.getTemplateItem(key)
	return template, errors.New("NoOp, item not found")
}

func (g NoOpReaderWriterLF[K, T]) Save(_ context.Context, _ T, _ any) error {
	if g.panicOnNext.CompareAndSwap(true, false) {
		panic("test panic, write")
	}
	msg := g.errorOnNext.Load()
	if msg != nil && strings.Contains(msg.(string), "save") {
		g.removeFromErrorOnNext()
		return errors.New("load " + msg.(string))
	}
	return nil
}

func (g NoOpReaderWriterLF[K, T]) BeginTx(_ context.Context) (tx any, err error) {
	if g.panicOnNext.CompareAndSwap(true, false) {
		panic("test panic, begin")
	}
	msg := g.errorOnNext.Load()
	if msg != nil && strings.Contains(msg.(string), "begin") {
		g.removeFromErrorOnNext()
		return nil, errors.New("beginTx " + msg.(string))
	}
	tx = "transaction"
	return tx, nil
}

func (g NoOpReaderWriterLF[K, T]) CommitTx(_ context.Context, _ any) error {
	if g.panicOnNext.CompareAndSwap(true, false) {
		panic("test panic, commit")
	}
	msg := g.errorOnNext.Load()
	if msg != nil && strings.Contains(msg.(string), "commit") {
		g.removeFromErrorOnNext()
		return errors.New("commitTx " + msg.(string))
	}
	return nil
}

func (g NoOpReaderWriterLF[K, T]) RollbackTx(_ context.Context, _ any) error {
	if g.panicOnNext.CompareAndSwap(true, false) {
		panic("test panic, rollback")
	}
	msg := g.errorOnNext.Load()
	if msg != nil && strings.Contains(msg.(string), "rollback") {
		g.removeFromErrorOnNext()
		return errors.New("rollbackTx " + msg.(string))
	}
	return nil
}

func (g NoOpReaderWriterLF[K, T]) Info(_ context.Context, msg string, _ CacheAction, _ ...T) {
	g.infoCount.Add(1)
	log.Print("[info] ", msg)
}

func (g NoOpReaderWriterLF[K, T]) Warn(_ context.Context, msg string, _ CacheAction, _ ...T) {
	g.warnCount.Add(1)
	log.Print("[warn] ", msg)
}

func (g NoOpReaderWriterLF[K, T]) PrintLog() {
	fmt.Println(g.logBuffer.String())
}

// remove first of comma separated list of errors.
func (g NoOpReaderWriterLF[K, T]) removeFromErrorOnNext() {
	next := g.errorOnNext.Load().(string)
	parts := strings.SplitN(next, ",", 2)
	if len(parts) == 2 {
		g.errorOnNext.Store(parts[1])
		// force a panic?
		if strings.Contains(parts[1], "panic") {
			g.errorOnNext.Store("")
			panic(parts[1])
		}
	} else {
		g.errorOnNext.Store("")
	}
}

func (g NoOpReaderWriterLF[K, T]) IsRecoverable(_ context.Context, err error) bool {
	if strings.Contains(strings.ToLower(err.Error()), "deadlock") {
		return true
	}
	if strings.Contains(strings.ToLower(err.Error()), "bad connection") { // mock for driver.ErrBadConn
		return true
	}
	return false
}

func (g NoOpReaderWriterLF[K, T]) Fail(_ context.Context, _ error, _ ...T) {
	g.fails.Add(1)
}
