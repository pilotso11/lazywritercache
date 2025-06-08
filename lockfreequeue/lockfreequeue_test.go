package lockfreequeue

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewLockFreeQueue(t *testing.T) {
	// Test creating a new queue
	q := NewLockFreeQueue[int]()
	assert.NotNil(t, q, "NewLockFreeQueue should return a non-nil queue")

	// Test that a new queue is empty (dequeue should return zero value)
	var zeroVal int
	val, ok := q.Dequeue()
	assert.False(t, ok, "Dequeue from empty queue should return false")
	assert.Equal(t, zeroVal, val, "Dequeue from empty queue should return zero value")
}

func TestEnqueueDequeue(t *testing.T) {
	q := NewLockFreeQueue[string]()

	// Test enqueue and dequeue operations
	q.Enqueue("first")
	q.Enqueue("second")
	q.Enqueue("third")

	// Test FIFO order
	item, ok := q.Dequeue()
	assert.True(t, ok, "Dequeue from non-empty queue should return true")
	assert.Equal(t, "first", item, "First dequeued item should be 'first'")
	item, ok = q.Dequeue()
	assert.True(t, ok, "Dequeue from non-empty queue should return true")
	assert.Equal(t, "second", item, "Second dequeued item should be 'second'")
	item, ok = q.Dequeue()
	assert.True(t, ok, "Dequeue from non-empty queue should return true")
	assert.Equal(t, "third", item, "Third dequeued item should be 'third'")

	// Test dequeue from empty queue
	var emptyString string
	item, ok = q.Dequeue()
	assert.False(t, ok, "Dequeue from empty queue should return false")
	assert.Equal(t, emptyString, item, "Dequeue from empty queue should return zero value")
}

func TestEnqueueDequeueWithDifferentTypes(t *testing.T) {
	// Test with int type
	qInt := NewLockFreeQueue[int]()
	qInt.Enqueue(42)
	qInt.Enqueue(100)
	item, ok := qInt.Dequeue()
	assert.True(t, ok, "Dequeue from non-empty queue should return true")
	assert.Equal(t, 42, item, "First dequeued int should be 42")
	item, ok = qInt.Dequeue()
	assert.True(t, ok, "Dequeue from non-empty queue should return true")
	assert.Equal(t, 100, item, "Second dequeued int should be 100")

	// Test with custom struct type
	type Person struct {
		Name string
		Age  int
	}

	qPerson := NewLockFreeQueue[Person]()
	alice := Person{Name: "Alice", Age: 30}
	bob := Person{Name: "Bob", Age: 25}

	qPerson.Enqueue(alice)
	qPerson.Enqueue(bob)

	firstPerson, ok := qPerson.Dequeue()
	assert.True(t, ok, "ok should be true for item in queue")
	secondPerson, ok := qPerson.Dequeue()
	assert.True(t, ok, "ok should be true for item in queue")

	assert.Equal(t, alice, firstPerson, "First dequeued person should be Alice")
	assert.Equal(t, bob, secondPerson, "Second dequeued person should be Bob")
}

func TestConcurrentOperations(t *testing.T) {
	q := NewLockFreeQueue[int]()
	const numGoroutines = 10
	const numOperations = 1000

	// Use a WaitGroup to ensure all goroutines complete
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Launch goroutines that concurrently enqueue items
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				q.Enqueue(id*numOperations + j)
			}
		}(i)
	}

	// Wait for all enqueue operations to complete
	wg.Wait()

	// Count the number of items we can dequeue
	count := 0
	for {
		val, _ := q.Dequeue()
		if val == 0 && count >= numGoroutines*numOperations {
			break
		}
		count++
		if count > numGoroutines*numOperations {
			t.Fatalf("Dequeued more items than expected: got %d, want %d", count, numGoroutines*numOperations)
		}
	}

	assert.Equal(t, numGoroutines*numOperations, count, "Should dequeue exactly the number of enqueued items")
}

func TestPeek(t *testing.T) {
	// Test with empty queue
	q1 := NewLockFreeQueue[int]()
	var zeroVal int
	val, ok := q1.Peek()
	assert.False(t, ok, "Peek from empty queue should return false")
	assert.Equal(t, zeroVal, val, "Peek from empty queue should return zero value")

	// Test with integers
	q2 := NewLockFreeQueue[int]()
	q2.Enqueue(1)
	q2.Enqueue(2)
	q2.Enqueue(3)

	// Peek should return the first item without removing it
	val, ok = q2.Peek()
	assert.True(t, ok, "Peek from non-empty queue should return true")
	assert.Equal(t, 1, val, "Peek should return the first item")

	// Peek again should return the same item
	val, ok = q2.Peek()
	assert.True(t, ok, "Peek from non-empty queue should return true")
	assert.Equal(t, 1, val, "Peek should return the first item again")

	// Dequeue should remove the first item
	val, ok = q2.Dequeue()
	assert.True(t, ok, "Dequeue from non-empty queue should return true")
	assert.Equal(t, 1, val, "Dequeue should return the first item")

	// Peek should now return the new first item
	val, ok = q2.Peek()
	assert.True(t, ok, "Peek from non-empty queue should return true")
	assert.Equal(t, 2, val, "Peek should return the new first item")

	// Test with custom struct type
	type Person struct {
		Name string
		Age  int
	}

	qPerson := NewLockFreeQueue[Person]()
	alice := Person{Name: "Alice", Age: 30}
	bob := Person{Name: "Bob", Age: 25}

	qPerson.Enqueue(alice)
	qPerson.Enqueue(bob)

	// Peek should return Alice
	firstPerson, ok := qPerson.Peek()
	assert.True(t, ok, "Peek from non-empty queue should return true")
	assert.Equal(t, alice, firstPerson, "Peek should return Alice")

	// Peek again should still return Alice
	firstPerson, ok = qPerson.Peek()
	assert.True(t, ok, "Peek from non-empty queue should return true")
	assert.Equal(t, alice, firstPerson, "Peek should still return Alice")

	// Dequeue should remove Alice
	firstPerson, ok = qPerson.Dequeue()
	assert.True(t, ok, "Dequeue from non-empty queue should return true")
	assert.Equal(t, alice, firstPerson, "Dequeue should return Alice")

	// Peek should now return Bob
	secondPerson, ok := qPerson.Peek()
	assert.True(t, ok, "Peek from non-empty queue should return true")
	assert.Equal(t, bob, secondPerson, "Peek should now return Bob")
}

func TestPeekWithTailFallingBehind(t *testing.T) {
	// Create a new queue
	q := NewLockFreeQueue[string]()

	// Enqueue an item
	q.Enqueue("test-item")

	// At this point, the queue structure should be:
	// head -> dummy -> node("test-item")
	// tail -> node("test-item")

	// Manually reset tail to point to the dummy node to simulate tail falling behind
	// This creates the scenario where head == tail but next != nil
	head := q.head.Load()
	q.tail.Store(head) // Now tail is pointing to the dummy node (falling behind)

	// Verify that the tail is now falling behind
	tail := q.tail.Load()
	assert.Equal(t, head, tail, "Head and tail should be the same after manipulation")
	assert.NotNil(t, head.next.Load(), "Next should not be nil, confirming tail is falling behind")

	// Call Peek() - it should handle the tail falling behind and return the correct item
	val, ok := q.Peek()
	assert.True(t, ok, "Peek should return true even when tail is falling behind")
	assert.Equal(t, "test-item", val, "Peek should return the correct item even when tail is falling behind")

	// Verify that the queue structure is corrected (tail is advanced)
	newTail := q.tail.Load()
	assert.NotEqual(t, head, newTail, "Tail should have been advanced by Peek()")

	// Verify that a subsequent Peek still returns the correct value
	val, ok = q.Peek()
	assert.True(t, ok, "Peek should still return true")
	assert.Equal(t, "test-item", val, "Peek should still return the correct item")
}

func TestString(t *testing.T) {
	// Test with empty queue
	q1 := NewLockFreeQueue[int]()
	assert.Equal(t, "[]", q1.String(), "Empty queue should be represented as '[]'")

	// Test with integers
	q2 := NewLockFreeQueue[int]()
	q2.Enqueue(1)
	q2.Enqueue(2)
	q2.Enqueue(3)
	assert.Equal(t, "[1, 2, 3]", q2.String(), "Queue with integers should be correctly represented")

	// Test with strings
	q3 := NewLockFreeQueue[string]()
	q3.Enqueue("apple")
	q3.Enqueue("banana")
	q3.Enqueue("cherry")
	assert.Equal(t, "[apple, banana, cherry]", q3.String(), "Queue with strings should be correctly represented")

	// Test that String() doesn't modify the queue
	item, ok := q3.Dequeue()
	assert.True(t, ok)
	assert.Equal(t, "apple", item, "First item should still be 'apple' after String() call")
	assert.Equal(t, "[banana, cherry]", q3.String(), "Queue should be correctly represented after dequeue")

	// Test with custom struct
	type Person struct {
		Name string
		Age  int
	}
	q4 := NewLockFreeQueue[Person]()
	q4.Enqueue(Person{Name: "Alice", Age: 30})
	q4.Enqueue(Person{Name: "Bob", Age: 25})
	// The exact string representation depends on how Go formats structs, so we just check it contains the names
	str := q4.String()
	assert.Contains(t, str, "Alice")
	assert.Contains(t, str, "Bob")
	assert.Contains(t, str, "30")
	assert.Contains(t, str, "25")
}

func TestConcurrentEnqueueDequeue(t *testing.T) {
	// Use a WaitGroup to ensure all goroutines complete
	var producerWg sync.WaitGroup
	var consumerWg sync.WaitGroup

	var queued, dequeued, pDone, cDone, producersDone atomic.Int64

	q := NewLockFreeQueue[int]()
	const numProducers = 4
	const numConsumers = 4
	const numItemsPerProducer = 1000

	// Start producer goroutines
	producerWg.Add(numProducers)
	for i := 0; i < numProducers; i++ {
		go func(id int) {
			defer producerWg.Done()
			defer pDone.Add(1)
			for j := 0; j < numItemsPerProducer; j++ {
				// Enqueue items with a unique value
				q.Enqueue(id*numItemsPerProducer + j + 1) // +1 to avoid zero values
				queued.Add(1)
			}
		}(i)
	}

	// Start consumer goroutines
	consumerWg.Add(numConsumers)
	for i := 0; i < numConsumers; i++ {
		go func() {
			defer consumerWg.Done()
			defer cDone.Add(1)
			for {
				_, ok := q.Dequeue()
				if ok {
					dequeued.Add(1)
				} else {
					if producersDone.Load() == 1 {
						return
					}
					time.Sleep(time.Millisecond) // pause for a bit and check again
				}
			}
		}()
	}

	var completed atomic.Int64
	go func() {
		// Wait for producers to finish
		producerWg.Wait()
		producersDone.Store(1)

		// Wait for consumers to finish
		consumerWg.Wait()

		completed.Store(1)
	}()

	assert.Eventuallyf(t, func() bool {
		return completed.Load() == 1
	}, 200*time.Millisecond, time.Millisecond, "should complete in 200ms")

	assert.Equal(t, int64(numProducers*numItemsPerProducer), dequeued.Load(), "Should dequeue exactly the number of enqueued items: ")
	assert.Equal(t, int64(1), producersDone.Load(), "queued: %d, dequeued: %d, pDone: %d, cDone: %d",
		queued.Load(), dequeued.Load(), pDone.Load(), cDone.Load())
}
