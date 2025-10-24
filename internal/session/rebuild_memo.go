// Package session provides session-scoped state management for goenv operations.
// This includes memoization of expensive operations to improve performance.
package session

import (
	"sync"
)

// RebuildMemo tracks which tools have been checked or rebuilt in this session.
// This prevents repeated architecture verification and rebuild prompts for the same tool.
type RebuildMemo struct {
	mu      sync.RWMutex
	checked map[string]bool // toolPath -> has been checked
	rebuilt map[string]bool // toolPath -> has been rebuilt
}

var (
	globalRebuildMemo *RebuildMemo
	memoOnce          sync.Once
)

// GetRebuildMemo returns the global rebuild memoization instance.
func GetRebuildMemo() *RebuildMemo {
	memoOnce.Do(func() {
		globalRebuildMemo = &RebuildMemo{
			checked: make(map[string]bool),
			rebuilt: make(map[string]bool),
		}
	})
	return globalRebuildMemo
}

// HasChecked returns true if this tool has already been checked in this session.
func (m *RebuildMemo) HasChecked(toolPath string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.checked[toolPath]
}

// MarkChecked marks a tool as having been checked for architecture compatibility.
func (m *RebuildMemo) MarkChecked(toolPath string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checked[toolPath] = true
}

// HasRebuilt returns true if this tool has already been rebuilt in this session.
func (m *RebuildMemo) HasRebuilt(toolPath string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.rebuilt[toolPath]
}

// MarkRebuilt marks a tool as having been rebuilt for the host architecture.
func (m *RebuildMemo) MarkRebuilt(toolPath string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rebuilt[toolPath] = true
}

// ShouldPromptRebuild returns true if we should prompt the user to rebuild this tool.
// Returns false if we've already rebuilt or prompted for this tool in this session.
func (m *RebuildMemo) ShouldPromptRebuild(toolPath string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Don't prompt if already rebuilt or already checked
	return !m.rebuilt[toolPath] && !m.checked[toolPath]
}

// Clear resets the memoization state (useful for testing).
func (m *RebuildMemo) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checked = make(map[string]bool)
	m.rebuilt = make(map[string]bool)
}
