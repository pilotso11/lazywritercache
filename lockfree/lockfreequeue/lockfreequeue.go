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

// Original from https://www.sobyte.net/post/2021-07/implementing-lock-free-queues-with-go/

package lockfreequeue

import (
	"sync/atomic"
	"unsafe"
)

// LockFreeQueue is a lock-free unbounded queue.
type LockFreeQueue[T any] struct {
	head unsafe.Pointer
	tail unsafe.Pointer
}
type node[T any] struct {
	value T
	next  unsafe.Pointer
}

// NewLockFreeQueue returns an empty queue.
func NewLockFreeQueue[T any]() *LockFreeQueue[T] {
	n := unsafe.Pointer(&node[T]{})
	return &LockFreeQueue[T]{head: n, tail: n}
}

// Enqueue puts the given value v at the tail of the queue.
func (q *LockFreeQueue[T]) Enqueue(v T) {
	n := &node[T]{value: v}
	for {
		tail := load[T](&q.tail)
		next := load[T](&tail.next)
		if tail == load[T](&q.tail) { // are tail and next consistent?
			if next == nil {
				if cas[T](&tail.next, next, n) {
					cas[T](&q.tail, tail, n) // Enqueue is done.  try to swing tail to the inserted node
					return
				}
			} else { // tail was not pointing to the last node
				// try to swing Tail to the next node
				cas(&q.tail, tail, next)
			}
		}
	}
}

// Dequeue removes and returns the value at the head of the queue.
// It returns nil if the queue is empty.
func (q *LockFreeQueue[T]) Dequeue() (val T) {
	for {
		head := load[T](&q.head)
		tail := load[T](&q.tail)
		next := load[T](&head.next)
		if head == load[T](&q.head) { // are head, tail, and next consistent?
			if head == tail { // is queue empty or tail falling behind?
				if next == nil { // is queue empty?
					return
				}
				// tail is falling behind.  try to advance it
				cas(&q.tail, tail, next)
			} else {
				// read value before CAS otherwise another dequeue might free the next node
				v := next.value
				if cas(&q.head, head, next) {
					return v // Dequeue is done.  return
				}
			}
		}
	}
}

func load[T any](p *unsafe.Pointer) (n *node[T]) {
	return (*node[T])(atomic.LoadPointer(p))
}

func cas[T any](p *unsafe.Pointer, old, new *node[T]) (ok bool) {
	return atomic.CompareAndSwapPointer(
		p, unsafe.Pointer(old), unsafe.Pointer(new))
}
