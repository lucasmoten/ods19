package audit

import (
	"fmt"
	"sync"

	auditevents "decipher.com/object-drive-server/services/audit/generated/events_thrift"
)

// NewPayloads initializes a Payloads struct.
func NewPayloads(size int) *Payloads {
	mp := make(map[string]*Queue)
	mtx := &sync.Mutex{}
	return &Payloads{M: mp, RequestArraySize: size, Mtx: mtx}
}

// Payloads buffers requests to audit services.
type Payloads struct {
	M                map[string]*Queue
	RequestArraySize int
	Mtx              *sync.Mutex
}

// DefaultMaxRequestArraySize limits the length of the slice returned
var DefaultMaxRequestArraySize = 10

// Add pushes an event to the internal Queue of a Payloads.
func (p *Payloads) Add(key string, event *auditevents.AuditEvent) {

	n := &Node{event}
	p.Mtx.Lock()
	_, ok := p.M[key]
	if !ok {
		// Queue not presetn for key. Create one.
		// We MUST initialize with at least length 1.
		q := &Queue{nodes: make([]*Node, 1)}
		p.M[key] = q
	}
	q, _ := p.M[key]
	q.Push(n)
	p.Mtx.Unlock()
	fmt.Println("Count of events:", q.count)
}

type Node struct {
	Value *auditevents.AuditEvent
}

// Stack is a basic LIFO stack that resizes as needed.
type Stack struct {
	nodes []*Node
	count int
}

// Push adds a node to the stack.
func (s *Stack) Push(n *Node) {
	if s.count >= len(s.nodes) {
		nodes := make([]*Node, len(s.nodes)*2)
		copy(nodes, s.nodes)
		s.nodes = nodes
	}
	s.nodes[s.count] = n
	s.count++
}

// Pop removes and returns a node from the stack in last to first order.
func (s *Stack) Pop() *Node {
	if s.count == 0 {
		return nil
	}
	node := s.nodes[s.count-1]
	s.count--
	return node
}

// Queue is a basic FIFO queue based on a circular list that resizes as needed.
type Queue struct {
	nodes []*Node
	head  int
	tail  int
	count int
}

// Pop removes and returns a node from the queue in first to last order.
func (q *Queue) Pop() *Node {
	if q.count == 0 {
		return nil
	}
	node := q.nodes[q.head]
	q.head = (q.head + 1) % len(q.nodes)
	q.count--
	return node
}

// Push adds a node to the queue.
func (q *Queue) Push(n *Node) {
	if q.head == q.tail && q.count > 0 {
		nodes := make([]*Node, len(q.nodes)*2)
		copy(nodes, q.nodes[q.head:])
		copy(nodes[len(q.nodes)-q.head:], q.nodes[:q.head])
		q.head = 0
		q.tail = len(q.nodes)
		q.nodes = nodes
	}
	q.nodes[q.tail] = n
	q.tail = (q.tail + 1) % len(q.nodes)
	q.count++
}
