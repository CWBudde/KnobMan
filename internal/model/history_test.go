package model

import "testing"

// helper: create a doc with a distinguishable width value.
func docWithWidth(w int) *Document {
	d := NewDocument()
	d.Prefs.PWidth.Val = w

	return d
}

func TestHistoryPushUndo(t *testing.T) {
	h := NewHistory()

	d1 := docWithWidth(10)
	d2 := docWithWidth(20)
	d3 := docWithWidth(30)

	h.Push(d1)
	h.Push(d2)
	h.Push(d3)

	got := h.Undo(docWithWidth(40))
	if got == nil || got.Prefs.PWidth.Val != 30 {
		t.Fatalf("first undo: want width 30, got %v", got)
	}

	got = h.Undo(got)
	if got == nil || got.Prefs.PWidth.Val != 20 {
		t.Fatalf("second undo: want width 20, got %v", got)
	}
}

func TestHistoryRedo(t *testing.T) {
	h := NewHistory()

	h.Push(docWithWidth(10))
	h.Push(docWithWidth(20))

	current := docWithWidth(30)

	prev := h.Undo(current)
	if prev == nil || prev.Prefs.PWidth.Val != 20 {
		t.Fatalf("undo: want width 20, got %v", prev)
	}

	next := h.Redo(prev)
	if next == nil || next.Prefs.PWidth.Val != 30 {
		t.Fatalf("redo: want width 30, got %v", next)
	}
}

func TestHistoryRedoClearedOnPush(t *testing.T) {
	h := NewHistory()

	h.Push(docWithWidth(10))
	h.Push(docWithWidth(20))

	current := docWithWidth(30)

	prev := h.Undo(current)
	if !h.CanRedo() {
		t.Fatal("expected CanRedo after undo")
	}

	// New push should clear the redo stack.
	h.Push(prev)

	if h.CanRedo() {
		t.Fatal("redo stack should be empty after new push")
	}
}

func TestHistoryMaxLen(t *testing.T) {
	h := NewHistory()
	for i := range 60 {
		h.Push(docWithWidth(i))
	}

	count := 0

	current := docWithWidth(999)
	for h.CanUndo() {
		current = h.Undo(current)
		count++
	}

	if count != defaultHistoryLen {
		t.Fatalf("expected %d undo steps, got %d", defaultHistoryLen, count)
	}
}

func TestHistoryUndoEmpty(t *testing.T) {
	h := NewHistory()
	if got := h.Undo(docWithWidth(1)); got != nil {
		t.Fatalf("undo on empty history should return nil, got %v", got)
	}
}

func TestHistoryRedoEmpty(t *testing.T) {
	h := NewHistory()
	if got := h.Redo(docWithWidth(1)); got != nil {
		t.Fatalf("redo on empty history should return nil, got %v", got)
	}
}
