package eventmesh

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"math"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Priority represents the Quality of Service (QoS) tier for an event.
type Priority int

const (
	// PriorityCritical is the highest priority tier. Never dropped under tail-drop load shedding.
	PriorityCritical Priority = iota
	// PriorityHigh is high priority. Dropped only under extreme queue saturation (>95%).
	PriorityHigh
	// PriorityDefault is standard priority. Subject to tail-drop shedding at 85% capacity.
	PriorityDefault
	// PriorityTelemetry is lowest priority (logs/metrics). First to be shed under load (>70%).
	PriorityTelemetry
)

func (p Priority) String() string {
	switch p {
	case PriorityCritical:
		return "CRITICAL"
	case PriorityHigh:
		return "HIGH"
	case PriorityDefault:
		return "DEFAULT"
	case PriorityTelemetry:
		return "TELEMETRY"
	default:
		return "UNKNOWN"
	}
}

// Event represents a structured domain event transmitted across the A2A mesh.
type Event struct {
	ID          string            
	Topic       string            
	Source      string            
	Priority    Priority          
	Payload     any               
	Headers     map[string]string 
	Timestamp   time.Time         
	SequenceNum uint64            
	RetryCount  int               
	Error       string            
}

// FilterFunc defines a custom predicate function for subscription filtering.
type FilterFunc func(event *Event) bool

// HandlerFunc defines the callback function invoked when a matching event arrives.
type HandlerFunc func(ctx context.Context, event *Event) error

// Common Errors
var (
	ErrEventDropped   = errors.New("event dropped due to tail-drop load shedding")
	ErrQueueFull      = errors.New("queue is full")
	ErrMeshClosed     = errors.New("event mesh is closed")
	ErrSubClosed      = errors.New("subscription is closed")
	ErrInvalidTopic   = errors.New("invalid topic format")
	ErrNilHandler     = errors.New("handler cannot be nil")
	ErrDLQNotFound    = errors.New("event not found in dead-letter queue")
)

// ringSlot represents a single slot in the lock-free MPMC ring buffer.
type ringSlot struct {
	event atomic.Pointer[Event]
	seq   atomic.Uint64
}

// LockFreeRingBuffer is a lock-free, multi-producer multi-consumer (MPMC) ring buffer.
// It uses Dmitry Vyukov's bounded MPMC queue design with atomic sequence checks.
type LockFreeRingBuffer struct {
	capacity uint64
	mask     uint64
	slots    []ringSlot
	head     atomic.Uint64
	tail     atomic.Uint64
	dropped  atomic.Uint64
}

// NewLockFreeRingBuffer initializes a lock-free MPMC ring buffer.
// Capacity will be rounded up to the nearest power of 2.
func NewLockFreeRingBuffer(capacity uint64) *LockFreeRingBuffer {
	if capacity < 2 {
		capacity = 2
	}
	// Round up to power of 2
	capacity = 1 << uint(math.Ceil(math.Log2(float64(capacity))))

	rb := &LockFreeRingBuffer{
		capacity: capacity,
		mask:     capacity - 1,
		slots:    make([]ringSlot, capacity),
	}
	for i := uint64(0); i < capacity; i++ {
		rb.slots[i].seq.Store(i)
	}
	return rb
}

// Capacity returns the total buffer capacity.
func (rb *LockFreeRingBuffer) Capacity() uint64 {
	return rb.capacity
}

// Size returns the approximate number of elements currently stored in the buffer.
func (rb *LockFreeRingBuffer) Size() uint64 {
	head := rb.head.Load()
	tail := rb.tail.Load()
	if tail >= head {
		return tail - head
	}
	return 0
}

// FillRatio returns the current buffer utilization ratio between 0.0 and 1.0.
func (rb *LockFreeRingBuffer) FillRatio() float64 {
	return float64(rb.Size()) / float64(rb.capacity)
}

// TryPush attempts to push an event into the lock-free ring buffer.
// Returns false if the buffer is full.
func (rb *LockFreeRingBuffer) TryPush(event *Event) bool {
	for {
		tail := rb.tail.Load()
		slot := &rb.slots[tail&rb.mask]
		seq := slot.seq.Load()
		dif := int64(seq) - int64(tail)

		if dif == 0 {
			if rb.tail.CompareAndSwap(tail, tail+1) {
				slot.event.Store(event)
				slot.seq.Store(tail + 1)
				return true
			}
		} else if dif < 0 {
			// Queue is full
			return false
		} else {
			// Another producer updated tail, retry
			runtime.Gosched()
		}
	}
}

// TryPop attempts to pop an event from the lock-free ring buffer.
// Returns nil, false if the buffer is empty.
func (rb *LockFreeRingBuffer) TryPop() (*Event, bool) {
	for {
		head := rb.head.Load()
		slot := &rb.slots[head&rb.mask]
		seq := slot.seq.Load()
		dif := int64(seq) - int64(head+1)

		if dif == 0 {
			if rb.head.CompareAndSwap(head, head+1) {
				event := slot.event.Swap(nil)
				slot.seq.Store(head + rb.mask + 1)
				return event, true
			}
		} else if dif < 0 {
			// Queue is empty
			return nil, false
		} else {
			// Another consumer updated head, retry
			runtime.Gosched()
		}
	}
}

// SubscriptionOptions configures a subscription.
type SubscriptionOptions struct {
	ID          string
	Filter      FilterFunc
	MinPriority Priority
	MaxRetries  int
	WorkerCount int
	BufferSize  uint64
}

// SubscriptionOption is a functional configuration option for subscriptions.
type SubscriptionOption func(*SubscriptionOptions)

func WithID(id string) SubscriptionOption {
	return func(opts *SubscriptionOptions) {
		opts.ID = id
	}
}

func WithFilter(filter FilterFunc) SubscriptionOption {
	return func(opts *SubscriptionOptions) {
		opts.Filter = filter
	}
}

func WithMinPriority(p Priority) SubscriptionOption {
	return func(opts *SubscriptionOptions) {
		opts.MinPriority = p
	}
}

func WithMaxRetries(retries int) SubscriptionOption {
	return func(opts *SubscriptionOptions) {
		opts.MaxRetries = retries
	}
}

func WithWorkerCount(workers int) SubscriptionOption {
	return func(opts *SubscriptionOptions) {
		opts.WorkerCount = workers
	}
}

func WithBufferSize(size uint64) SubscriptionOption {
	return func(opts *SubscriptionOptions) {
		opts.BufferSize = size
	}
}

// Subscription represents an active subscriber listening on a topic pattern.
type Subscription struct {
	id          string
	pattern     string
	handler     HandlerFunc
	filter      FilterFunc
	minPriority Priority
	maxRetries  int
	buffer      *LockFreeRingBuffer
	notifyChan  chan struct{}
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	closed      atomic.Bool
	mesh        *EventMesh
}

// EventMesh is the central in-memory A2A Event Mesh engine.
type EventMesh struct {
	subMu          sync.RWMutex
	subs           map[string]*Subscription
	topicIndex     map[string][]*Subscription
	dlqMu          sync.RWMutex
	dlqMap         map[string]*Event
	dlqList        []*Event
	dlqCapacity    int
	seqCounter     atomic.Uint64
	closed         atomic.Bool
	ctx            context.Context
	cancel         context.CancelFunc
	globalBuffer   *LockFreeRingBuffer
	globalNotify   chan struct{}
	globalWg       sync.WaitGroup
	metrics        Metrics
}

// Metrics tracks real-time statistics for the event mesh.
type Metrics struct {
	PublishedEvents    atomic.Uint64
	DeliveredEvents    atomic.Uint64
	DroppedTelemetry   atomic.Uint64
	DroppedDefault     atomic.Uint64
	DroppedHigh        atomic.Uint64
	DLQCount           atomic.Uint64
	ActiveSubsCount    atomic.Uint64
}

// NewEventMesh creates and initializes a new A2A Event Mesh instance.
func NewEventMesh(bufferSize uint64, dlqCapacity int) *EventMesh {
	if bufferSize == 0 {
		bufferSize = 4096
	}
	if dlqCapacity <= 0 {
		dlqCapacity = 1000
	}

	ctx, cancel := context.WithCancel(context.Background())
	em := &EventMesh{
		subs:         make(map[string]*Subscription),
		topicIndex:   make(map[string][]*Subscription),
		dlqMap:       make(map[string]*Event),
		dlqCapacity:  dlqCapacity,
		ctx:          ctx,
		cancel:       cancel,
		globalBuffer: NewLockFreeRingBuffer(bufferSize),
		globalNotify: make(chan struct{}, 1),
	}

	// Start global event dispatcher
	em.globalWg.Add(1)
	go em.runGlobalDispatcher()

	return em
}

// Publish emits a new event onto the A2A mesh with adaptive backpressure and tail-drop load shedding.
func (em *EventMesh) Publish(event *Event) error {
	if em.closed.Load() {
		return ErrMeshClosed
	}

	if event.Topic == "" {
		return ErrInvalidTopic
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	event.SequenceNum = em.seqCounter.Add(1)
	if event.ID == "" {
		event.ID = fmt.Sprintf("evt-%d-%d", event.Timestamp.UnixNano(), event.SequenceNum)
	}

	// Evaluate Adaptive Backpressure & Tail-Drop Load Shedding
	fillRatio := em.globalBuffer.FillRatio()

	switch event.Priority {
	case PriorityTelemetry:
		// Drop telemetry early if fill ratio >= 70%
		if fillRatio >= 0.70 {
			em.metrics.DroppedTelemetry.Add(1)
			return ErrEventDropped
		}
	case PriorityDefault:
		// Tail-drop default messages when capacity reaches 85%
		if fillRatio >= 0.85 {
			em.metrics.DroppedDefault.Add(1)
			return ErrEventDropped
		}
	case PriorityHigh:
		// Drop high priority messages only under extreme saturation >= 95%
		if fillRatio >= 0.95 {
			em.metrics.DroppedHigh.Add(1)
			return ErrEventDropped
		}
	case PriorityCritical:
		// Critical messages are NEVER tail-dropped.
		// If queue is 100% full, spin briefly until slot opens.
	}

	// Try pushing into global lock-free ring buffer
	pushed := em.globalBuffer.TryPush(event)
	if !pushed {
		if event.Priority == PriorityCritical {
			// For Critical events, spin-wait with Gosched until accepted
			for !pushed && !em.closed.Load() {
				runtime.Gosched()
				pushed = em.globalBuffer.TryPush(event)
			}
			if em.closed.Load() {
				return ErrMeshClosed
			}
		} else {
			// For other priorities, increment drop counter
			switch event.Priority {
			case PriorityHigh:
				em.metrics.DroppedHigh.Add(1)
			case PriorityDefault:
				em.metrics.DroppedDefault.Add(1)
			case PriorityTelemetry:
				em.metrics.DroppedTelemetry.Add(1)
			}
			return ErrQueueFull
		}
	}

	em.metrics.PublishedEvents.Add(1)

	// Notify dispatcher non-blockingly
	select {
	case em.globalNotify <- struct{}{}:
	default:
	}

	return nil
}

// Subscribe registers a new subscriber for a given topic pattern.
func (em *EventMesh) Subscribe(topicPattern string, handler HandlerFunc, opts ...SubscriptionOption) (*Subscription, error) {
	if em.closed.Load() {
		return nil, ErrMeshClosed
	}

	if handler == nil {
		return nil, ErrNilHandler
	}

	if topicPattern == "" {
		return nil, ErrInvalidTopic
	}

	sOpts := SubscriptionOptions{
		ID:          fmt.Sprintf("sub-%d", time.Now().UnixNano()),
		MinPriority: PriorityTelemetry,
		MaxRetries:  3,
		WorkerCount: 4,
		BufferSize:  1024,
	}

	for _, opt := range opts {
		opt(&sOpts)
	}

	subCtx, subCancel := context.WithCancel(em.ctx)

	sub := &Subscription{
		id:          sOpts.ID,
		pattern:     topicPattern,
		handler:     handler,
		filter:      sOpts.Filter,
		minPriority: sOpts.MinPriority,
		maxRetries:  sOpts.MaxRetries,
		buffer:      NewLockFreeRingBuffer(sOpts.BufferSize),
		notifyChan:  make(chan struct{}, 1),
		ctx:         subCtx,
		cancel:      subCancel,
		mesh:        em,
	}

	em.subMu.Lock()
	em.subs[sub.id] = sub
	em.rebuildTopicIndex()
	em.metrics.ActiveSubsCount.Store(uint64(len(em.subs)))
	em.subMu.Unlock()

	// Start subscriber worker pool
	for i := 0; i < sOpts.WorkerCount; i++ {
		sub.wg.Add(1)
		go sub.workerLoop()
	}

	return sub, nil
}

// Unsubscribe removes a subscription from the event mesh.
func (em *EventMesh) Unsubscribe(subID string) error {
	em.subMu.Lock()
	sub, exists := em.subs[subID]
	if !exists {
		em.subMu.Unlock()
		return fmt.Errorf("subscription %s not found", subID)
	}

	delete(em.subs, subID)
	em.rebuildTopicIndex()
	em.metrics.ActiveSubsCount.Store(uint64(len(em.subs)))
	em.subMu.Unlock()

	sub.close()
	return nil
}

// rebuildTopicIndex rebuilds topic lookup mappings (called under subMu write lock).
func (em *EventMesh) rebuildTopicIndex() {
	em.topicIndex = make(map[string][]*Subscription)
	for _, sub := range em.subs {
		em.topicIndex[sub.pattern] = append(em.topicIndex[sub.pattern], sub)
	}
}

// runGlobalDispatcher consumes events from the global ring buffer and dispatches to matching subscribers.
func (em *EventMesh) runGlobalDispatcher() {
	defer em.globalWg.Done()

	for {
		select {
		case <-em.ctx.Done():
			return
		case <-em.globalNotify:
			em.dispatchPendingEvents()
		}
	}
}

// dispatchPendingEvents pops events from global buffer and routes to subscribers.
func (em *EventMesh) dispatchPendingEvents() {
	for {
		event, ok := em.globalBuffer.TryPop()
		if !ok {
			break
		}

		em.subMu.RLock()
		for _, sub := range em.subs {
			if matchTopic(sub.pattern, event.Topic) {
				if event.Priority > sub.minPriority {
					continue
				}
				if sub.filter != nil && !sub.filter(event) {
					continue
				}
				sub.enqueue(event)
			}
		}
		em.subMu.RUnlock()
	}
}

// enqueue pushes an event into a subscription's buffer and notifies workers.
func (s *Subscription) enqueue(event *Event) {
	if s.closed.Load() {
		return
	}

	// Apply subscription buffer capacity check
	fillRatio := s.buffer.FillRatio()
	if event.Priority == PriorityDefault && fillRatio >= 0.85 {
		s.mesh.metrics.DroppedDefault.Add(1)
		return
	}
	if event.Priority == PriorityTelemetry && fillRatio >= 0.70 {
		s.mesh.metrics.DroppedTelemetry.Add(1)
		return
	}

	if s.buffer.TryPush(event) {
		select {
		case s.notifyChan <- struct{}{}:
		default:
		}
	}
}

// workerLoop processes events for a subscription.
func (s *Subscription) workerLoop() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.notifyChan:
			s.processBuffer()
		}
	}
}

func (s *Subscription) processBuffer() {
	for {
		event, ok := s.buffer.TryPop()
		if !ok {
			break
		}

		err := s.handler(s.ctx, event)
		if err != nil {
			event.RetryCount++
			if event.RetryCount <= s.maxRetries {
				// Re-enqueue for retry
				s.enqueue(event)
			} else {
				// Exceeded retries -> send to Dead-Letter Queue (DLQ)
				event.Error = fmt.Sprintf("failed after %d retries: %v", event.RetryCount, err)
				s.mesh.recordDLQ(event)
			}
		} else {
			s.mesh.metrics.DeliveredEvents.Add(1)
		}
	}
}

func (s *Subscription) close() {
	if s.closed.CompareAndSwap(false, true) {
		s.cancel()
		s.wg.Wait()
	}
}

// DLQ Handling
func (em *EventMesh) recordDLQ(event *Event) {
	em.dlqMu.Lock()
	defer em.dlqMu.Unlock()

	if len(em.dlqList) >= em.dlqCapacity {
		// Remove oldest
		oldest := em.dlqList[0]
		delete(em.dlqMap, oldest.ID)
		em.dlqList = em.dlqList[1:]
	}

	em.dlqMap[event.ID] = event
	em.dlqList = append(em.dlqList, event)
	em.metrics.DLQCount.Store(uint64(len(em.dlqList)))
}

// GetDLQEvents returns failed events matching an optional topic filter.
func (em *EventMesh) GetDLQEvents(topic string) []*Event {
	em.dlqMu.RLock()
	defer em.dlqMu.RUnlock()

	res := make([]*Event, 0, len(em.dlqList))
	for _, ev := range em.dlqList {
		if topic == "" || matchTopic(topic, ev.Topic) {
			res = append(res, ev)
		}
	}
	return res
}

// ReplayDLQ re-publishes a failed event from the DLQ by ID.
func (em *EventMesh) ReplayDLQ(eventID string) error {
	em.dlqMu.Lock()
	ev, exists := em.dlqMap[eventID]
	if !exists {
		em.dlqMu.Unlock()
		return ErrDLQNotFound
	}

	// Remove from DLQ map/list
	delete(em.dlqMap, eventID)
	newList := make([]*Event, 0, len(em.dlqList)-1)
	for _, item := range em.dlqList {
		if item.ID != eventID {
			newList = append(newList, item)
		}
	}
	em.dlqList = newList
	em.metrics.DLQCount.Store(uint64(len(em.dlqList)))
	em.dlqMu.Unlock()

	// Reset error & retries before re-publishing
	ev.Error = ""
	ev.RetryCount = 0
	return em.Publish(ev)
}

// AllDLQ returns a Go 1.23 iterator over all DLQ events.
func (em *EventMesh) AllDLQ() iter.Seq[*Event] {
	return func(yield func(*Event) bool) {
		em.dlqMu.RLock()
		events := make([]*Event, len(em.dlqList))
		copy(events, em.dlqList)
		em.dlqMu.RUnlock()

		for _, ev := range events {
			if !yield(ev) {
				return
			}
		}
	}
}

// GetMetrics returns a snapshot of metrics.
func (em *EventMesh) GetMetrics() map[string]uint64 {
	return map[string]uint64{
		"published_events":  em.metrics.PublishedEvents.Load(),
		"delivered_events":  em.metrics.DeliveredEvents.Load(),
		"dropped_telemetry": em.metrics.DroppedTelemetry.Load(),
		"dropped_default":   em.metrics.DroppedDefault.Load(),
		"dropped_high":      em.metrics.DroppedHigh.Load(),
		"dlq_count":         em.metrics.DLQCount.Load(),
		"active_subs":       em.metrics.ActiveSubsCount.Load(),
		"queue_size":        em.globalBuffer.Size(),
	}
}

// Close gracefully shuts down the Event Mesh.
func (em *EventMesh) Close() {
	if em.closed.CompareAndSwap(false, true) {
		em.cancel()

		em.subMu.Lock()
		for id, sub := range em.subs {
			sub.close()
			delete(em.subs, id)
		}
		em.subMu.Unlock()

		em.globalWg.Wait()
	}
}

// Topic Pattern Matcher (supports exact, wildcard '*' and multi-wildcard '>')
func matchTopic(pattern, topic string) bool {
	if pattern == ">" || pattern == topic {
		return true
	}

	pTokens := strings.Split(pattern, ".")
	tTokens := strings.Split(topic, ".")

	for i := 0; i < len(pTokens); i++ {
		p := pTokens[i]
		if p == ">" {
			return true
		}

		if i >= len(tTokens) {
			return false
		}

		if p != "*" && p != tTokens[i] {
			return false
		}
	}

	return len(pTokens) == len(tTokens)
}
