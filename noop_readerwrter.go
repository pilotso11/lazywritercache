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
	"sync"
	// No "sync/atomic" import
)

type NoOpReaderWriter[T Cacheable] struct {
	getTemplateItem    func(key interface{}) T
	mockedItems        map[string]T
	succeedOnFind      map[string]bool
	errorOnSave        map[string]error
	panicOnLoad        bool
	panicOnWrite       bool
	panicOnSaveKey     string
	beginTxError       error
	commitTxPanic      bool
	deadlockErrorCount map[string]int
	deadlockRetries    map[string]int

	PostSaveCallback func(key string)
	FindCallCount    int64
	SaveCallCount    map[string]int64
	WarnCallCount    int64
	WarnMessages     []string
	InfoCallCount    int64
	InfoMessages     []string
	mu               sync.Mutex
}

var _ CacheReaderWriter[string, EmptyCacheable] = (*NoOpReaderWriter[EmptyCacheable])(nil)

func NewNoOpReaderWriter[T Cacheable](itemTemplate func(key any) T, forcePanics ...bool) *NoOpReaderWriter[T] {
	doPanics := len(forcePanics) > 0 && forcePanics[0]
	return &NoOpReaderWriter[T]{
		getTemplateItem:    itemTemplate,
		mockedItems:        make(map[string]T),
		succeedOnFind:      make(map[string]bool),
		errorOnSave:        make(map[string]error),
		panicOnLoad:        doPanics,
		panicOnWrite:       doPanics,
		deadlockErrorCount: make(map[string]int),
		deadlockRetries:    make(map[string]int),
		SaveCallCount:      make(map[string]int64),
		WarnMessages:       make([]string, 0),
		InfoMessages:       make([]string, 0),
	}
}

func (rw *NoOpReaderWriter[T]) SetMockedItem(key string, item T) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	rw.mockedItems[key] = item
}

func (rw *NoOpReaderWriter[T]) SetSucceedOnFind(key string, succeed bool) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	rw.succeedOnFind[key] = succeed
}

func (rw *NoOpReaderWriter[T]) SetErrorOnSave(key string, err error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	rw.errorOnSave[key] = err
}

func (rw *NoOpReaderWriter[T]) SetBeginTxError(err error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	rw.beginTxError = err
}

func (rw *NoOpReaderWriter[T]) SetCommitTxPanic(panicFlag bool) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	rw.commitTxPanic = panicFlag
}

func (rw *NoOpReaderWriter[T]) SimulateDeadlockOnSave(key string, retries int) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	rw.deadlockRetries[key] = retries
	rw.errorOnSave[key] = errors.New("simulated deadlock error")
}

func (rw *NoOpReaderWriter[T]) SetPanicOnSaveKey(key string) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	rw.panicOnSaveKey = key
}

func (rw *NoOpReaderWriter[T]) SetPanicOnLoad(flag bool) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	rw.panicOnLoad = flag
}

func (rw *NoOpReaderWriter[T]) SetPanicOnWrite(flag bool) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	rw.panicOnWrite = flag
}
func (rw *NoOpReaderWriter[T]) SetPostSaveCallback(cb func(key string)){
	rw.mu.Lock()
	defer rw.mu.Unlock()
	rw.PostSaveCallback = cb
}


func (g *NoOpReaderWriter[T]) Find(key string, _ interface{}) (T, error) {
	g.mu.Lock()
	g.FindCallCount++
	// Reading panicOnLoad, succeedOnFind, mockedItems also needs protection if they can be modified concurrently by Setters.
	// Current Setters have mutex, so reads here should be fine if setters are not called mid-operation by same test goroutine.
	// However, for general safety, critical reads accessed by multiple methods should be protected.
	panicOnLoad := g.panicOnLoad
	succeed, specified := g.succeedOnFind[key]
	var item T
	var ok bool
	if specified && succeed {
		item, ok = g.mockedItems[key]
	}
	g.mu.Unlock() // Unlock after accessing shared fields

	if panicOnLoad {
		panic("test panic, read")
	}

	if specified && succeed {
		if ok {
			return item, nil
		}
		return g.getTemplateItem(key), nil
	}
	return g.getTemplateItem(key), errors.New("NoOp, item not found")
}

func (g *NoOpReaderWriter[T]) Save(item T, _ interface{}) error {
	keyStr, iok := item.Key().(string)
	if !iok {
		log.Printf("Warning: Item key is not a string in NoOpReaderWriter.Save for test setup. Key: %v", item.Key())
		panic("Item key is not a string in NoOpReaderWriter.Save for test setup")
	}

	g.mu.Lock()
	// All checks and modifications of shared fields under one lock acquisition
	if g.panicOnSaveKey != "" && keyStr == g.panicOnSaveKey {
		g.mu.Unlock()
		panic("test panic, write on specific key: " + keyStr)
	}
	if g.panicOnWrite {
		g.mu.Unlock()
		panic("test panic, write")
	}

	g.SaveCallCount[keyStr]++

	if retries, active := g.deadlockRetries[keyStr]; active {
		currentRetries := g.deadlockErrorCount[keyStr]
		currentRetries++
		g.deadlockErrorCount[keyStr] = currentRetries
		if currentRetries <= retries {
			errToRet := g.errorOnSave[keyStr]
			g.mu.Unlock()
			return errToRet
		}
		delete(g.deadlockRetries, keyStr)
		if val, ok := g.errorOnSave[keyStr]; ok && val.Error() == "simulated deadlock error" {
			delete(g.errorOnSave, keyStr)
		}

		cb := g.PostSaveCallback
		g.mu.Unlock() // Unlock before callback
		if cb != nil {
			cb(keyStr)
		}
		return nil
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

func (g *NoOpReaderWriter[T]) BeginTx() (tx interface{}, err error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.beginTxError != nil {
		return nil, g.beginTxError
	}
	return "transaction", nil
}

func (g *NoOpReaderWriter[T]) CommitTx(_ interface{}) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.commitTxPanic {
		panic("mock CommitTx panic")
	}
}

func (g *NoOpReaderWriter[T]) Info(msg string, _ string, _ ...T) {
	g.mu.Lock()
	g.InfoCallCount++
	g.InfoMessages = append(g.InfoMessages, msg)
	g.mu.Unlock()
}

func (g *NoOpReaderWriter[T]) Warn(msg string, _ string, _ ...T) {
	g.mu.Lock()
	g.WarnCallCount++
	g.WarnMessages = append(g.WarnMessages, msg)
	g.mu.Unlock()
}

func (rw *NoOpReaderWriter[T]) ResetCountersAndMessages() {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	rw.FindCallCount = 0
	rw.SaveCallCount = make(map[string]int64)
	rw.WarnCallCount = 0
	rw.WarnMessages = make([]string, 0)
	rw.InfoCallCount = 0
	rw.InfoMessages = make([]string, 0)
}
