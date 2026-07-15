package session

import (
	"sync"
)

// ContainerTracker tracks which queries are in which containers.
// Ported from td/td/telegram/net/Session.h:82-104 (Query with container_message_id_).
type ContainerTracker struct {
	containers       map[int64]*ContainerEntry
	childToContainer map[int64]int64
	logger           sessionLogger
	mu               sync.Mutex
}

// SetLogger sets the logger for the container tracker.
func (t *ContainerTracker) SetLogger(l sessionLogger) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.logger = l
}

// ContainerEntry tracks a container and its children.
type ContainerEntry struct {
	ContainerMsgID int64
	ChildMsgIDs    []int64
	RefCount       int
}

// NewContainerTracker creates a new container tracker.
func NewContainerTracker() *ContainerTracker {
	return &ContainerTracker{
		containers:       make(map[int64]*ContainerEntry),
		childToContainer: make(map[int64]int64),
	}
}

// TrackContainer registers a container with its children.
func (t *ContainerTracker) TrackContainer(containerMsgID int64, childMsgIDs []int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.containers[containerMsgID] = &ContainerEntry{
		ContainerMsgID: containerMsgID,
		ChildMsgIDs:    childMsgIDs,
		RefCount:       len(childMsgIDs),
	}
	for _, id := range childMsgIDs {
		t.childToContainer[id] = containerMsgID
	}
	if t.logger != nil {
		t.logger.Warnf("container tracked container_msg_id=%d children=%d", containerMsgID, len(childMsgIDs))
	}
}

// AckContainer ACKs all children of a container. Returns the child message IDs.
func (t *ContainerTracker) AckContainer(containerMsgID int64) []int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	entry, ok := t.containers[containerMsgID]
	if !ok {
		return nil
	}
	childIDs := entry.ChildMsgIDs
	delete(t.containers, containerMsgID)
	for _, id := range childIDs {
		delete(t.childToContainer, id)
	}
	return childIDs
}

// NackContainer NACKs all children of a container. Returns the child message IDs.
func (t *ContainerTracker) NackContainer(containerMsgID int64) []int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	entry, ok := t.containers[containerMsgID]
	if !ok {
		return nil
	}
	childIDs := entry.ChildMsgIDs
	if t.logger != nil {
		t.logger.Warnf("container nacked container_msg_id=%d children=%d", containerMsgID, len(childIDs))
	}
	delete(t.containers, containerMsgID)
	for _, id := range childIDs {
		delete(t.childToContainer, id)
	}
	return childIDs
}

// AckChild ACKs a single child, decrementing the ref count.
// Returns true if the container is now fully ACKed.
func (t *ContainerTracker) AckChild(childMsgID int64) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	containerMsgID, ok := t.childToContainer[childMsgID]
	if !ok {
		return false
	}
	entry := t.containers[containerMsgID]
	if entry == nil {
		return false
	}
	entry.RefCount--
	if t.logger != nil {
		t.logger.Warnf("container child acked child_msg_id=%d container_msg_id=%d remaining=%d", childMsgID, containerMsgID, entry.RefCount)
	}
	if entry.RefCount <= 0 {
		delete(t.containers, containerMsgID)
		for _, id := range entry.ChildMsgIDs {
			delete(t.childToContainer, id)
		}
		return true
	}
	return false
}

// Cleanup removes all entries (called on session close).
func (t *ContainerTracker) Cleanup() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.containers = make(map[int64]*ContainerEntry)
	t.childToContainer = make(map[int64]int64)
}

// Count returns the number of tracked containers.
func (t *ContainerTracker) Count() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.containers)
}
