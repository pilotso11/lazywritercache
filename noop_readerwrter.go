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
	// stdAtomic "sync/atomic" // No longer needed
)

type NoOpReaderWriter[T Cacheable] struct {
	getTemplateItem    func(key interface{}) T
	mockedItems        map[string]T
	succeedOnFind      map[string]bool
	errorOnSave        map[string]error
	panicOnLoad        bool
	panicOnWrite       bool // Global panic on write
	panicOnSaveKey     string // Specific key to cause panic on Save
	beginTxError       error
	commitTxPanic      bool
	deadlockErrorCount map[string]int
	deadlockRetries    map[string]int

	// For testing specific scenarios
	PostSaveCallback func(key string)
	FindCallCount    int64 // Changed to int64
	SaveCallCount    map[string]int64 // Changed map value to int64
	WarnCallCount    int64 // Changed to int64
	WarnMessages     []string
	InfoCallCount    int64 // Changed to int64
	InfoMessages     []string
	mu               sync.Mutex
}

// Check interface is complete
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
		SaveCallCount:      make(map[string]int64), // Initialize with int64 values
		// FindCallCount, WarnCallCount, InfoCallCount are zero by default for int64
		WarnMessages:       make([]string, 0),
		InfoMessages:       make([]string, 0),
	}
}

func (rw *NoOpReaderWriter[T]) SetMockedItem(key string, item T) {
	rw.mockedItems[key] = item
}

func (rw *NoOpReaderWriter[T]) SetSucceedOnFind(key string, succeed bool) {
	rw.succeedOnFind[key] = succeed
}

func (rw *NoOpReaderWriter[T]) SetErrorOnSave(key string, err error) {
	rw.errorOnSave[key] = err
}

func (rw *NoOpReaderWriter[T]) SetBeginTxError(err error) {
	rw.beginTxError = err
}

func (rw *NoOpReaderWriter[T]) SetCommitTxPanic(panicFlag bool) {
	rw.commitTxPanic = panicFlag
}

func (rw *NoOpReaderWriter[T]) SimulateDeadlockOnSave(key string, retries int) {
	rw.deadlockRetries[key] = retries
	// Ensure the error message contains "deadlock" for the cache logic to pick it up.
	rw.errorOnSave[key] = errors.New("simulated deadlock error")
}

func (rw *NoOpReaderWriter[T]) SetPanicOnSaveKey(key string) {
	rw.panicOnSaveKey = key
}

func (g *NoOpReaderWriter[T]) Find(key string, _ interface{}) (T, error) {
	g.mu.Lock()
	g.FindCallCount++
	g.mu.Unlock()
	if g.panicOnLoad {
		panic("test panic, read")
	}
	succeed, specified := g.succeedOnFind[key]
	if specified && succeed {
		item, ok := g.mockedItems[key]
		if ok {
			return item, nil
		}
		// If SetSucceedOnFind was true, but no item was mocked, return template (as if found but empty)
		return g.getTemplateItem(key), nil
	}
	return g.getTemplateItem(key), errors.New("NoOp, item not found")
}

func (g *NoOpReaderWriter[T]) Save(item T, _ interface{}) error {
	keyStr, ok := item.Key().(string)
	if !ok {
		// Fallback or error if key cannot be converted to string.
		// This depends on how strictly test items adhere to string keys.
		// For now, let's assume test keys will be strings.
		log.Printf("Warning: Item key is not a string in NoOpReaderWriter.Save for test setup. Key: %v", item.Key())
		// Attempt to proceed if possible, or return an error
		// For testing, it might be better to panic if the key isn't string,
		// as it indicates a test setup issue.
		panic("Item key is not a string in NoOpReaderWriter.Save for test setup")
	}

	if g.panicOnSaveKey != "" && keyStr == g.panicOnSaveKey {
		panic("test panic, write on specific key: " + keyStr)
	}
	if g.panicOnWrite { // Global panic if specific key not matched or not set
		panic("test panic, write")
	}

	g.mu.Lock()
	g.SaveCallCount[keyStr]++
	g.mu.Unlock()
	// The duplicated block below was removed.
	// if !ok {
	// log.Printf("Warning: Item key is not a string in NoOpReaderWriter.Save for test setup. Key: %v", item.Key())
	// panic("Item key is not a string in NoOpReaderWriter.Save for test setup")
	// }
	// g.SaveCallCount[keyStr]++ // this was the second increment, also removed.

	if retries, active := g.deadlockRetries[keyStr]; active {
		currentRetries := g.deadlockErrorCount[keyStr]
		currentRetries++
		g.deadlockErrorCount[keyStr] = currentRetries
		if currentRetries <= retries {
			// Return the error set by SimulateDeadlockOnSave, which should contain "deadlock"
			return g.errorOnSave[keyStr]
		}
		delete(g.deadlockRetries, keyStr)
		// If deadlock was the only error for this key, remove it from errorOnSave too
		// This logic assumes SimulateDeadlockOnSave sets errorOnSave.
		if val, ok := g.errorOnSave[keyStr]; ok && val.Error() == "simulated deadlock error" {
			delete(g.errorOnSave, keyStr)
		}
		// Successful save after retries
		if g.PostSaveCallback != nil {
			g.PostSaveCallback(keyStr)
		}
		return nil
	}

	if err, specified := g.errorOnSave[keyStr]; specified {
		return err
	}

	if g.PostSaveCallback != nil {
		g.PostSaveCallback(keyStr)
	}
	return nil
}

func (g *NoOpReaderWriter[T]) BeginTx() (tx interface{}, err error) {
	if g.beginTxError != nil {
		return nil, g.beginTxError
	}
	return "transaction", nil
}

func (g *NoOpReaderWriter[T]) CommitTx(_ interface{}) {
	if g.commitTxPanic {
		panic("mock CommitTx panic")
	}
}

func (g *NoOpReaderWriter[T]) Info(msg string, _ string, _ ...T) {
	g.mu.Lock()
	g.InfoCallCount++
	g.InfoMessages = append(g.InfoMessages, msg)
	g.mu.Unlock()
	// log.Print("[info] ", msg) // Original log.Print can be noisy for tests
}

func (g *NoOpReaderWriter[T]) Warn(msg string, _ string, _ ...T) {
	g.mu.Lock()
	g.WarnCallCount++
	g.WarnMessages = append(g.WarnMessages, msg)
	g.mu.Unlock()
	// log.Print("[warn] ", msg) // Original log.Print can be noisy for tests
}

// Helper to reset call counts and messages for fresh assertions in tests
func (rw *NoOpReaderWriter[T]) ResetCountersAndMessages() {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	rw.FindCallCount = 0
	rw.SaveCallCount = make(map[string]int64)
	rw.WarnCallCount = 0
	rw.WarnMessages = make([]string, 0)
	rw.InfoCallCount = 0
	rw.InfoMessages = make([]string, 0)
	// Do not reset error configurations or mocked items by default
}
