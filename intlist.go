package main

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

const INT_MAX = int(^uint(0) >> 1)
const INT_MIN = ^INT_MAX

// LinkedList wrapper of nodes, it's a singly linked-list with header node
type LinkedList struct {
	head *linkedListNode
	len  uint32
}

// linkedListNode the each node of list
type linkedListNode struct {
	next  *linkedListNode // next next node of list
	key   int             // key stored key
	isDel uint32          // atomic operation
	m     sync.Mutex      // lock for write
}

// newLinkedListNode create a new node of node
func newLinkedListNode(next *linkedListNode, key int) *linkedListNode {
	return &linkedListNode{
		next, key, 0, sync.Mutex{},
	}
}

// NewLinkedList create a new LinkedList with a INT_MIN head
func NewLinkedList() *LinkedList {
	return &LinkedList{newLinkedListNode(nil, INT_MIN), 0}
}

// NewInt alias for test
func NewInt() *LinkedList {
	return &LinkedList{newLinkedListNode(nil, INT_MIN), 0}
}

// Contains return if the list contains the needle
func (l *LinkedList) Contains(needle int) bool {
	_, next := l.find(needle)
	return next != nil && next.key == needle && atomic.LoadUint32(&next.isDel) == 0
}

// Insert insert a key to list
func (l *LinkedList) Insert(key int) bool {
	cur, next := l.find(key)
	// check if the node is exist
	if next != nil && next.key == key {
		return false
	}
	cur.m.Lock()
	if cur.next != next || cur.isDel == 1 {
		cur.m.Unlock()
		return l.Insert(key) // recursive retry, the defer is unavailable
	}
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&cur.next)), unsafe.Pointer(newLinkedListNode(next, key)))
	cur.m.Unlock()
	atomic.AddUint32(&l.len, 1)
	return true
}

// Delete delete a key from list
func (l *LinkedList) Delete(key int) bool {
	cur, next := l.find(key)
	if next == nil || next.key != key {
		return false
	}
	next.m.Lock()
	if next.isDel == 1 {
		next.m.Unlock()
		return l.Delete(key) // recursive retry, the defer is unavailable
	}
	cur.m.Lock()
	if cur.next != next || cur.isDel == 1 {
		cur.m.Unlock()
		next.m.Unlock()
		return l.Delete(key) // recursive retry, the defer is unavailable
	}
	atomic.StoreUint32(&next.isDel, 1)
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&cur.next)), unsafe.Pointer(next.next))
	cur.m.Unlock()
	next.m.Unlock()
	atomic.AddUint32(&l.len, ^uint32(0))
	return true
}

// Range travels the list
func (l *LinkedList) Range(f func(value int) bool) {
	x := (*linkedListNode)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&l.head.next))))
	for x != nil && f(x.key) {
		x = (*linkedListNode)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&x.next))))
	}
}

// Len return the len of list
func (l *LinkedList) Len() int {
	return int(atomic.LoadUint32(&l.len))
}

// find return two nodes which may contain the interval of needle
func (l *LinkedList) find(needle int) (cur, next *linkedListNode) {
	cur = l.head
	next = (*linkedListNode)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&l.head.next))))
	for next != nil && next.key < needle {
		cur = next
		next = (*linkedListNode)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&next.next))))
	}
	return
}
