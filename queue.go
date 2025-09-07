package main

import (
	"container/list"
	"context"
	"fmt"
	"sync"
	"time"

	pb "github.com/adammck/collector/proto/gen"
)

type QueueItem struct {
	ID        string
	Request   *pb.Request
	Response  chan *pb.Response
	AddedAt   time.Time
	Deferred  bool
	Context   context.Context
}

type QueueStatus struct {
	Total    int `json:"total"`
	Active   int `json:"active"`
	Deferred int `json:"deferred"`
}

type Queue struct {
	items    *list.List
	itemsMap map[string]*list.Element
	mu       sync.RWMutex

	waiters map[chan struct{}]struct{}
	wmu     sync.Mutex
}

func NewQueue() *Queue {
	return &Queue{
		items:    list.New(),
		itemsMap: make(map[string]*list.Element),
		waiters:  make(map[chan struct{}]struct{}),
	}
}

func (q *Queue) Enqueue(item *QueueItem) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, exists := q.itemsMap[item.ID]; exists {
		return fmt.Errorf("item already in queue: %s", item.ID)
	}

	elem := q.items.PushBack(item)
	q.itemsMap[item.ID] = elem
	q.notifyWaiters()

	return nil
}

func (q *Queue) Dequeue() (*QueueItem, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for e := q.items.Front(); e != nil; e = e.Next() {
		item := e.Value.(*QueueItem)

		if !item.Deferred {
			q.items.Remove(e)
			delete(q.itemsMap, item.ID)
			return item, nil
		}
	}

	return nil, fmt.Errorf("queue empty or all items deferred")
}

func (q *Queue) Defer(id string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	elem, ok := q.itemsMap[id]
	if !ok {
		return fmt.Errorf("item not found: %s", id)
	}

	item := elem.Value.(*QueueItem)
	item.Deferred = true

	q.items.MoveToBack(elem)

	return nil
}

func (q *Queue) Remove(id string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	elem, ok := q.itemsMap[id]
	if !ok {
		return fmt.Errorf("item not found: %s", id)
	}

	q.items.Remove(elem)
	delete(q.itemsMap, id)

	return nil
}

func (q *Queue) Status() QueueStatus {
	q.mu.RLock()
	defer q.mu.RUnlock()

	active := 0
	deferred := 0

	for e := q.items.Front(); e != nil; e = e.Next() {
		if e.Value.(*QueueItem).Deferred {
			deferred++
		} else {
			active++
		}
	}

	return QueueStatus{
		Total:    q.items.Len(),
		Active:   active,
		Deferred: deferred,
	}
}

func (q *Queue) GetNext(timeout time.Duration) (*QueueItem, error) {
	ch := make(chan struct{})

	q.wmu.Lock()
	q.waiters[ch] = struct{}{}
	q.wmu.Unlock()

	defer func() {
		q.wmu.Lock()
		delete(q.waiters, ch)
		q.wmu.Unlock()
	}()

	timeoutCh := time.After(timeout)

	for {
		item, err := q.Dequeue()
		if err == nil {
			return item, nil
		}

		select {
		case <-ch:
			continue
		case <-timeoutCh:
			return nil, fmt.Errorf("timeout waiting for queue item")
		}
	}
}

func (q *Queue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.items.Init()
	q.itemsMap = make(map[string]*list.Element)
}

func (q *Queue) notifyWaiters() {
	q.wmu.Lock()
	defer q.wmu.Unlock()

	for ch := range q.waiters {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}