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

package lockfree

import (
	"errors"
	"sync"
	"sync/atomic"
)

type NoOpReaderWriterLF[T CacheableLF] struct {
	getTemplateItem  func(key string) T
	mockedItems      map[string]T
	succeedOnFind    map[string]bool
	errorOnSave      map[string]error
	panicOnLoad      bool
	panicOnWrite     bool   // Global panic on write
	panicOnSaveKey   string // Specific key to cause panic on Save
	beginTxError     error
	commitTxPanic    bool
	FindCallCount    atomic.Int64
	SaveCallCount    map[string]*atomic.Int64 // Keyed by item key
	WarnCallCount    atomic.Int64
	WarnMessages     []string
	InfoCallCount    atomic.Int64
	InfoMessages     []string
	PostSaveCallback func(key string)
	mu               sync.Mutex // For thread-safe access to slices like WarnMessages and map SaveCallCount
}

// Check interface is complete
var _ CacheReaderWriterLF[EmptyCacheableLF] = (*NoOpReaderWriterLF[EmptyCacheableLF])(nil)

func NewNoOpReaderWriterLF[T CacheableLF](itemTemplate func(key string) T, forcePanics ...bool) *NoOpReaderWriterLF[T] {
	doPanics := len(forcePanics) > 0 && forcePanics[0]
	return &NoOpReaderWriterLF[T]{
		getTemplateItem: itemTemplate,
		mockedItems:     make(map[string]T),
		succeedOnFind:   make(map[string]bool),
		errorOnSave:     make(map[string]error),
		panicOnLoad:     doPanics,
		panicOnWrite:    doPanics,
		SaveCallCount:   make(map[string]*atomic.Int64),
		WarnMessages:    []string{},
		InfoMessages:    []string{},
	}
}

func (rw *NoOpReaderWriterLF[T]) ResetCountersAndMessages() {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	rw.FindCallCount.Store(0)
	rw.SaveCallCount = make(map[string]*atomic.Int64) // Reset map
	rw.WarnCallCount.Store(0)
	rw.WarnMessages = []string{}
	rw.InfoCallCount.Store(0)
	rw.InfoMessages = []string{}
}

func (rw *NoOpReaderWriterLF[T]) GetSaveCount(key string) int64 {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	if count, ok := rw.SaveCallCount[key]; ok {
		return count.Load()
	}
	return 0
}

func (rw *NoOpReaderWriterLF[T]) SetMockedItem(key string, item T) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	rw.mockedItems[key] = item
}

func (rw *NoOpReaderWriterLF[T]) SetSucceedOnFind(key string, succeed bool) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	rw.succeedOnFind[key] = succeed
}

func (rw *NoOpReaderWriterLF[T]) SetErrorOnSave(key string, err error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	rw.errorOnSave[key] = err
}

func (rw *NoOpReaderWriterLF[T]) SetBeginTxError(err error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	rw.beginTxError = err
}

func (rw *NoOpReaderWriterLF[T]) SetCommitTxPanic(panicFlag bool) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	rw.commitTxPanic = panicFlag
}

func (rw *NoOpReaderWriterLF[T]) SetPanicOnLoad(flag bool) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	rw.panicOnLoad = flag
}

func (rw *NoOpReaderWriterLF[T]) SetPanicOnWrite(flag bool) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	rw.panicOnWrite = flag
}

func (rw *NoOpReaderWriterLF[T]) SetPostSaveCallback(cb func(key string)) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	rw.PostSaveCallback = cb
}

func (rw *NoOpReaderWriterLF[T]) SetPanicOnSaveKey(key string) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	rw.panicOnSaveKey = key
}

func (g *NoOpReaderWriterLF[T]) Find(key string, _ interface{}) (T, error) {
	g.FindCallCount.Add(1)
	g.mu.Lock() // Lock for reading panicOnLoad and maps
	if g.panicOnLoad {
		g.mu.Unlock()
		panic("test panic, read")
	}
	succeed, specified := g.succeedOnFind[key]
	if specified && succeed {
		item, ok := g.mockedItems[key]
		g.mu.Unlock() // Unlock before returning from this path
		if ok {
			return item, nil
		}
		return g.getTemplateItem(key), nil
	}
	g.mu.Unlock() // Unlock if not returned above
	return g.getTemplateItem(key), errors.New("NoOp, item not found")
}

func (g *NoOpReaderWriterLF[T]) Save(item T, _ interface{}) error {
	keyStr := item.Key()

	g.mu.Lock() // Lock for map access and panicOnWrite
	if _, ok := g.SaveCallCount[keyStr]; !ok {
		g.SaveCallCount[keyStr] = &atomic.Int64{}
	}
	g.SaveCallCount[keyStr].Add(1)

	if g.panicOnSaveKey != "" && keyStr == g.panicOnSaveKey {
		g.mu.Unlock()
		panic("test panic, write on specific key: " + keyStr)
	}
	if g.panicOnWrite { // Global panic if specific key not matched or not set
		g.mu.Unlock()
		panic("test panic, write")
	}

	if err, specified := g.errorOnSave[keyStr]; specified {
		g.mu.Unlock()
		return err
	}

	cb := g.PostSaveCallback
	g.mu.Unlock() // Unlock before callback

	if cb != nil {
		cb(keyStr)
	}
	return nil
}

func (g *NoOpReaderWriterLF[T]) BeginTx() (tx interface{}, err error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.beginTxError != nil {
		return nil, g.beginTxError
	}
	return "transaction", nil
}

func (g *NoOpReaderWriterLF[T]) CommitTx(_ interface{}) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.commitTxPanic {
		panic("mock CommitTx panic")
	}
}

func (g *NoOpReaderWriterLF[T]) Info(msg string, action string, item ...T) {
	g.InfoCallCount.Add(1)
	g.mu.Lock()
	defer g.mu.Unlock()
	g.InfoMessages = append(g.InfoMessages, msg)
	// log.Print("[info] ", msg) // Original log.Print can be noisy for tests
}

func (g *NoOpReaderWriterLF[T]) Warn(msg string, action string, item ...T) {
	g.WarnCallCount.Add(1)
	g.mu.Lock()
	defer g.mu.Unlock()
	g.WarnMessages = append(g.WarnMessages, msg)
	// log.Print("[warn] ", msg) // Original log.Print can be noisy for tests
}
