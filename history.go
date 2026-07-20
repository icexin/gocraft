package main

import "sync"

type EditAction struct {
	Position Vec3
	OldType  int
	NewType  int
}

type EditHistory struct {
	mu      sync.Mutex
	stack   []EditAction
	maxSize int
}

func NewEditHistory(maxSize int) *EditHistory {
	return &EditHistory{
		maxSize: maxSize,
	}
}

func (h *EditHistory) Push(action EditAction) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.stack) >= h.maxSize {
		h.stack = h.stack[1:]
	}
	h.stack = append(h.stack, action)
}

func (h *EditHistory) Pop() (EditAction, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.stack) == 0 {
		return EditAction{}, false
	}
	action := h.stack[len(h.stack)-1]
	h.stack = h.stack[:len(h.stack)-1]
	return action, true
}

func (h *EditHistory) Len() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.stack)
}

func (h *EditHistory) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.stack = nil
}
