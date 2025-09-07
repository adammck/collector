package main

import (
	"fmt"
	"sync"
	"testing"
	"time"

	pb "github.com/adammck/collector/proto/gen"
)

func TestQueueBasicOperations(t *testing.T) {
	q := NewQueue()

	// test empty queue
	status := q.Status()
	if status.Total != 0 || status.Active != 0 || status.Deferred != 0 {
		t.Fatalf("expected empty queue, got: %+v", status)
	}

	// test enqueue
	item := &QueueItem{
		ID:       "test1",
		Request:  newTestRequest(),
		Response: make(chan *pb.Response, 1),
		AddedAt:  time.Now(),
	}

	if err := q.Enqueue(item); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	status = q.Status()
	if status.Total != 1 || status.Active != 1 || status.Deferred != 0 {
		t.Fatalf("expected 1 active item, got: %+v", status)
	}

	// test dequeue
	dequeued, err := q.Dequeue()
	if err != nil {
		t.Fatalf("dequeue failed: %v", err)
	}

	if dequeued.ID != "test1" {
		t.Fatalf("expected test1, got: %s", dequeued.ID)
	}

	// queue should be empty now
	status = q.Status()
	if status.Total != 0 {
		t.Fatalf("expected empty queue, got: %+v", status)
	}
}

func TestQueueFIFOOrdering(t *testing.T) {
	q := NewQueue()

	// enqueue multiple items
	items := []string{"first", "second", "third"}
	for _, id := range items {
		item := &QueueItem{
			ID:       id,
			Request:  newTestRequest(),
			Response: make(chan *pb.Response, 1),
			AddedAt:  time.Now(),
		}
		if err := q.Enqueue(item); err != nil {
			t.Fatalf("enqueue %s failed: %v", id, err)
		}
	}

	// dequeue should return in fifo order
	for _, expectedID := range items {
		item, err := q.Dequeue()
		if err != nil {
			t.Fatalf("dequeue failed: %v", err)
		}
		if item.ID != expectedID {
			t.Fatalf("expected %s, got %s", expectedID, item.ID)
		}
	}
}

func TestQueueDeferOperation(t *testing.T) {
	q := NewQueue()

	// enqueue three items
	items := []string{"first", "second", "third"}
	for _, id := range items {
		item := &QueueItem{
			ID:       id,
			Request:  newTestRequest(),
			Response: make(chan *pb.Response, 1),
			AddedAt:  time.Now(),
		}
		if err := q.Enqueue(item); err != nil {
			t.Fatalf("enqueue %s failed: %v", id, err)
		}
	}

	// defer the first item
	if err := q.Defer("first"); err != nil {
		t.Fatalf("defer failed: %v", err)
	}

	status := q.Status()
	if status.Total != 3 || status.Active != 2 || status.Deferred != 1 {
		t.Fatalf("expected 2 active, 1 deferred, got: %+v", status)
	}

	// dequeue should skip deferred item
	item, err := q.Dequeue()
	if err != nil {
		t.Fatalf("dequeue failed: %v", err)
	}
	if item.ID != "second" {
		t.Fatalf("expected second, got %s", item.ID)
	}

	// next dequeue should get third
	item, err = q.Dequeue()
	if err != nil {
		t.Fatalf("dequeue failed: %v", err)
	}
	if item.ID != "third" {
		t.Fatalf("expected third, got %s", item.ID)
	}

	// only deferred item should remain
	status = q.Status()
	if status.Total != 1 || status.Active != 0 || status.Deferred != 1 {
		t.Fatalf("expected 1 deferred item, got: %+v", status)
	}
}

func TestQueueConcurrentAccess(t *testing.T) {
	q := NewQueue()
	const numWorkers = 10
	const itemsPerWorker = 100

	var wg sync.WaitGroup

	// concurrent enqueues
	wg.Add(numWorkers)
	for w := 0; w < numWorkers; w++ {
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < itemsPerWorker; i++ {
				item := &QueueItem{
					ID:       fmt.Sprintf("worker%d-item%d", workerID, i),
					Request:  newTestRequest(),
					Response: make(chan *pb.Response, 1),
					AddedAt:  time.Now(),
				}
				if err := q.Enqueue(item); err != nil {
					t.Errorf("enqueue failed: %v", err)
					return
				}
			}
		}(w)
	}
	wg.Wait()

	status := q.Status()
	expected := numWorkers * itemsPerWorker
	if status.Total != expected {
		t.Fatalf("expected %d items, got %d", expected, status.Total)
	}

	// concurrent dequeues
	results := make(chan string, expected)
	wg.Add(numWorkers)
	for w := 0; w < numWorkers; w++ {
		go func() {
			defer wg.Done()
			for i := 0; i < itemsPerWorker; i++ {
				item, err := q.Dequeue()
				if err != nil {
					t.Errorf("dequeue failed: %v", err)
					return
				}
				results <- item.ID
			}
		}()
	}
	wg.Wait()
	close(results)

	// verify all items were dequeued
	count := 0
	for range results {
		count++
	}
	if count != expected {
		t.Fatalf("expected %d results, got %d", expected, count)
	}
}

func TestQueueGetNextWithTimeout(t *testing.T) {
	q := NewQueue()

	// test timeout on empty queue
	start := time.Now()
	_, err := q.GetNext(100 * time.Millisecond)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error")
	}

	if elapsed < 100*time.Millisecond {
		t.Fatalf("timeout too fast: %v", elapsed)
	}

	// test successful get
	item := &QueueItem{
		ID:       "test1",
		Request:  newTestRequest(),
		Response: make(chan *pb.Response, 1),
		AddedAt:  time.Now(),
	}

	if err := q.Enqueue(item); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	retrieved, err := q.GetNext(1 * time.Second)
	if err != nil {
		t.Fatalf("GetNext failed: %v", err)
	}

	if retrieved.ID != "test1" {
		t.Fatalf("expected test1, got %s", retrieved.ID)
	}
}