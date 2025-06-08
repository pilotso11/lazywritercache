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
	"fmt"
	"strings"
	"sync/atomic"
)

// LockFreeQueue is a lock-free unbounded queue.
type LockFreeQueue[T any] struct {
	head atomic.Pointer[node[T]]
	tail atomic.Pointer[node[T]]
}
type node[T any] struct {
	value T
	next  atomic.Pointer[node[T]]
}

// NewLockFreeQueue returns an empty queue.
func NewLockFreeQueue[T any]() *LockFreeQueue[T] {
	var n atomic.Pointer[node[T]]
	n.Store(&node[T]{})
	return &LockFreeQueue[T]{head: n, tail: n}
}

// Enqueue puts the given value v at the tail of the queue.
func (q *LockFreeQueue[T]) Enqueue(v T) {
	n := &node[T]{value: v}
	for {
		tail := q.tail.Load()
		next := tail.next.Load()
		if tail == q.tail.Load() { // are tail and next consistent?
			if next == nil {
				if tail.next.CompareAndSwap(next, n) { // try to link node at the end of the linked list
					q.tail.CompareAndSwap(tail, n) // Enqueue is done.  try to swing tail to the inserted node
					return
				}
			} else { // tail was not pointing to the last node
				// try to swing Tail to the next node
				q.tail.CompareAndSwap(tail, next)
			}
		}
	}
}

// Dequeue removes and returns the value at the head of the queue.
// It returns nil if the queue is empty.
func (q *LockFreeQueue[T]) Dequeue() (val T, ok bool) {
	for {
		head := q.head.Load()
		tail := q.tail.Load()
		next := head.next.Load()
		if head == q.head.Load() { // are head, tail, and next consistent?
			if head == tail { // is queue empty or tail falling behind?
				if next == nil { // is queue empty?
					return val, false
				}
				// tail is falling behind.  try to advance it
				q.tail.CompareAndSwap(tail, next)
			} else {
				// read value before CAS otherwise another dequeue might free the next node
				v := next.value
				if q.head.CompareAndSwap(head, next) {
					return v, true // Dequeue is done.  return
				}
			}
		}
	}
}

// Peek returns the value at the head of the queue without removing it.
// It returns the value and a boolean indicating whether the queue is empty.
func (q *LockFreeQueue[T]) Peek() (val T, ok bool) {
	for {
		head := q.head.Load()
		tail := q.tail.Load()
		next := head.next.Load()
		if head == q.head.Load() { // are head, tail, and next consistent?
			if head == tail { // is queue empty or tail falling behind?
				if next == nil { // is queue empty?
					return val, false
				}
				// tail is falling behind. try to advance it
				q.tail.CompareAndSwap(tail, next)
			} else {
				// read value and return it without modifying the queue
				return next.value, true
			}
		}
	}
}

// String returns a string representation of the queue's contents without modifying the queue.
// The items are displayed in order from head to tail.
func (q *LockFreeQueue[T]) String() string {
	var sb strings.Builder
	sb.WriteString("[")

	// Start from the first real node (after the dummy head node)
	current := q.head.Load().next.Load()
	isFirst := true

	// Traverse the queue
	for current != nil {
		if !isFirst {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("%v", current.value))
		isFirst = false
		current = current.next.Load()
	}

	sb.WriteString("]")
	return sb.String()
}
