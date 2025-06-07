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
	"strings"
	"sync/atomic"
)

type NoOpReaderWriter[T Cacheable] struct {
	getTemplateItem func(key interface{}) T
	errorOnNext     *atomic.Value
}

// Check interface is complete
var _ CacheReaderWriter[string, EmptyCacheable] = (*NoOpReaderWriter[EmptyCacheable])(nil)

func NewNoOpReaderWriter[T Cacheable](itemTemplate func(key any) T) NoOpReaderWriter[T] {
	w := NoOpReaderWriter[T]{
		getTemplateItem: itemTemplate,
		errorOnNext:     &atomic.Value{},
	}
	w.errorOnNext.Store("")
	return w
}

func (g NoOpReaderWriter[T]) Find(key string, _ interface{}) (T, error) {
	template := g.getTemplateItem(key)
	if err := g.nextError("find"); err != nil {
		return template, err
	}
	return template, errors.New("NoOp, item not found")
}

func (g NoOpReaderWriter[T]) Save(_ T, _ interface{}) error {
	if err := g.nextError("save"); err != nil {
		return err
	}
	return nil
}

func (g NoOpReaderWriter[T]) BeginTx() (tx interface{}, err error) {
	if err := g.nextError("begin"); err != nil {
		return nil, err
	}
	tx = "transaction"
	return tx, nil
}

func (g NoOpReaderWriter[T]) CommitTx(_ interface{}) error {
	if err := g.nextError("commit"); err != nil {
		return err
	}
	return nil
}

func (g NoOpReaderWriter[T]) RollbackTx(_ interface{}) error {
	if err := g.nextError("rollback"); err != nil {
		return err
	}
	return nil
}

func (g NoOpReaderWriter[T]) Info(msg string, _ string, _ ...T) {
	log.Print("[info] ", msg)
}

func (g NoOpReaderWriter[T]) Warn(msg string, _ string, _ ...T) {
	log.Print("[warn] ", msg)
}

func (g NoOpReaderWriter[T]) nextError(s string) error {
	next := g.errorOnNext.Load().(string)
	if strings.Contains(next, s) {
		parts := strings.SplitN(next, ",", 2)
		err := errors.New(parts[0])
		if strings.Contains(parts[0], "panic") {
			panic(parts[0])
		}
		if len(parts) == 2 {
			g.errorOnNext.Store(parts[1])
		} else {
			g.errorOnNext.Store("")
		}
		return err
	}
	return nil

}
