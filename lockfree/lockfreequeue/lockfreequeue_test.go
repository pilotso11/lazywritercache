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
	val := q.Dequeue()
	assert.Equal(t, zeroVal, val, "Dequeue from empty queue should return zero value")
}

func TestEnqueueDequeue(t *testing.T) {
	q := NewLockFreeQueue[string]()

	// Test enqueue and dequeue operations
	q.Enqueue("first")
	q.Enqueue("second")
	q.Enqueue("third")

	// Test FIFO order
	assert.Equal(t, "first", q.Dequeue(), "First dequeued item should be 'first'")
	assert.Equal(t, "second", q.Dequeue(), "Second dequeued item should be 'second'")
	assert.Equal(t, "third", q.Dequeue(), "Third dequeued item should be 'third'")

	// Test dequeue from empty queue
	var emptyString string
	assert.Equal(t, emptyString, q.Dequeue(), "Dequeue from empty queue should return zero value")
}

func TestEnqueueDequeueWithDifferentTypes(t *testing.T) {
	// Test with int type
	qInt := NewLockFreeQueue[int]()
	qInt.Enqueue(42)
	qInt.Enqueue(100)
	assert.Equal(t, 42, qInt.Dequeue(), "First dequeued int should be 42")
	assert.Equal(t, 100, qInt.Dequeue(), "Second dequeued int should be 100")

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

	firstPerson := qPerson.Dequeue()
	secondPerson := qPerson.Dequeue()

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
		val := q.Dequeue()
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
				val := q.Dequeue()
				if val != 0 {
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
