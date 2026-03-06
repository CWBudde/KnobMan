package model

const defaultHistoryLen = 50

// History provides a simple linear undo/redo stack of Document snapshots.
type History struct {
	past   []*Document
	future []*Document
	maxLen int
}

// NewHistory returns an empty History with the default depth limit.
func NewHistory() *History {
	return &History{maxLen: defaultHistoryLen}
}

// Push saves the current document state before a mutation. Call this
// before every change that should be undoable.
func (h *History) Push(doc *Document) {
	h.past = append(h.past, doc.Clone())
	if len(h.past) > h.maxLen {
		h.past = h.past[len(h.past)-h.maxLen:]
	}

	h.future = h.future[:0] // new action clears redo stack
}

// Undo returns the previous document state and pushes the current one onto
// the redo stack. Returns nil if there is nothing to undo.
func (h *History) Undo(current *Document) *Document {
	if len(h.past) == 0 {
		return nil
	}

	h.future = append(h.future, current.Clone())
	prev := h.past[len(h.past)-1]
	h.past = h.past[:len(h.past)-1]

	return prev
}

// Redo returns the next document state. Returns nil if there is nothing to redo.
func (h *History) Redo(current *Document) *Document {
	if len(h.future) == 0 {
		return nil
	}

	h.past = append(h.past, current.Clone())
	next := h.future[len(h.future)-1]
	h.future = h.future[:len(h.future)-1]

	return next
}

// CanUndo reports whether an undo step is available.
func (h *History) CanUndo() bool { return len(h.past) > 0 }

// CanRedo reports whether a redo step is available.
func (h *History) CanRedo() bool { return len(h.future) > 0 }
