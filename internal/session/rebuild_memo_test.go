package session

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRebuildMemo_SingletonBehavior(t *testing.T) {
	memo1 := GetRebuildMemo()
	memo2 := GetRebuildMemo()

	assert.Equal(t, memo2, memo1, "GetRebuildMemo should return the same instance (singleton)")
}

func TestRebuildMemo_HasChecked(t *testing.T) {
	memo := GetRebuildMemo()
	memo.Clear() // Start fresh

	toolPath := "/test/tool/path"

	if memo.HasChecked(toolPath) {
		t.Error("Tool should not be checked initially")
	}

	memo.MarkChecked(toolPath)

	assert.True(t, memo.HasChecked(toolPath), "Tool should be checked after MarkChecked")
}

func TestRebuildMemo_HasRebuilt(t *testing.T) {
	memo := GetRebuildMemo()
	memo.Clear() // Start fresh

	toolPath := "/test/tool/rebuilt"

	if memo.HasRebuilt(toolPath) {
		t.Error("Tool should not be rebuilt initially")
	}

	memo.MarkRebuilt(toolPath)

	assert.True(t, memo.HasRebuilt(toolPath), "Tool should be rebuilt after MarkRebuilt")
}

func TestRebuildMemo_ShouldPromptRebuild(t *testing.T) {
	memo := GetRebuildMemo()
	memo.Clear() // Start fresh

	toolPath := "/test/tool/prompt"

	// Should prompt initially
	assert.True(t, memo.ShouldPromptRebuild(toolPath), "Should prompt for rebuild initially")

	// After checking, should not prompt
	memo.MarkChecked(toolPath)
	if memo.ShouldPromptRebuild(toolPath) {
		t.Error("Should not prompt after being checked")
	}

	// Reset and test with rebuild
	memo.Clear()
	toolPath2 := "/test/tool/prompt2"

	assert.True(t, memo.ShouldPromptRebuild(toolPath2), "Should prompt for rebuild initially")

	// After rebuilding, should not prompt
	memo.MarkRebuilt(toolPath2)
	if memo.ShouldPromptRebuild(toolPath2) {
		t.Error("Should not prompt after being rebuilt")
	}
}

func TestRebuildMemo_Clear(t *testing.T) {
	memo := GetRebuildMemo()
	memo.Clear() // Start fresh

	toolPath1 := "/test/tool/clear1"
	toolPath2 := "/test/tool/clear2"

	memo.MarkChecked(toolPath1)
	memo.MarkRebuilt(toolPath2)

	assert.False(t, !memo.HasChecked(toolPath1) || !memo.HasRebuilt(toolPath2), "Tools should be marked before clear")

	memo.Clear()

	if memo.HasChecked(toolPath1) {
		t.Error("Tool should not be checked after clear")
	}

	if memo.HasRebuilt(toolPath2) {
		t.Error("Tool should not be rebuilt after clear")
	}
}

func TestRebuildMemo_ConcurrentAccess(t *testing.T) {
	memo := GetRebuildMemo()
	memo.Clear() // Start fresh

	var wg sync.WaitGroup
	numGoroutines := 100

	// Test concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			toolPath := "/test/tool/concurrent"
			memo.MarkChecked(toolPath)
			memo.HasChecked(toolPath)
		}(i)
	}

	wg.Wait()

	// Tool should be marked as checked
	assert.True(t, memo.HasChecked("/test/tool/concurrent"), "Tool should be checked after concurrent access")
}

func TestRebuildMemo_MultipleTools(t *testing.T) {
	memo := GetRebuildMemo()
	memo.Clear() // Start fresh

	tools := []string{
		"/test/tool/a",
		"/test/tool/b",
		"/test/tool/c",
	}

	// Mark different states for different tools
	memo.MarkChecked(tools[0])
	memo.MarkRebuilt(tools[1])
	// tools[2] is untouched

	assert.True(t, memo.HasChecked(tools[0]), "Tool 0 should be checked")

	assert.True(t, memo.HasRebuilt(tools[1]), "Tool 1 should be rebuilt")

	assert.False(t, memo.HasChecked(tools[2]) || memo.HasRebuilt(tools[2]), "Tool 2 should not be checked or rebuilt")

	assert.True(t, memo.ShouldPromptRebuild(tools[2]), "Should prompt for tool 2")

	assert.False(t, memo.ShouldPromptRebuild(tools[0]) || memo.ShouldPromptRebuild(tools[1]), "Should not prompt for tools 0 or 1")
}
