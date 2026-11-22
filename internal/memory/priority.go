package memory

import "strings"

// The order of live tasks. The index, the panel and the pending report all list tasks
// in it, so what goes first is decided here, once.
//
// It depends on the notes only, never on today: MEMORY.md is compared with the index
// built again (doctor, the panel), and an order that moved with the date would make it
// look stale every morning. So "the deadline is near" is not part of it: a task with a
// deadline goes before one without, the nearest first, and the session start says
// which deadlines come in the next days.

// HighImportance is the one value of importance that changes anything; the rest,
// missing included, is normal.
const HighImportance = "high"

// High reports whether the task is marked as important.
func (h *Header) High() bool { return strings.EqualFold(h.Importance, HighImportance) }

// DueDate returns the task's deadline, and false when it has none or it is not a date.
func (h *Header) DueDate() (string, bool) {
	if h.Due == "" {
		return "", false
	}
	if _, err := ParseDate(h.Due); err != nil {
		return "", false
	}
	return h.Due, true
}

// rank is the tier of a task in the order: a deadline first, then importance, then
// waiting on the user, then the rest.
func rank(h *Header) int {
	switch _, due := h.DueDate(); {
	case due:
		return 0
	case h.High():
		return 1
	case h.WaitingOn == "me":
		return 2
	}
	return 3
}

// CompareTasks orders two tasks: those with a deadline first (the nearest first), then
// the important ones, then those waiting on the user, then the rest; within each, the
// one untouched for longest first, and then by id, so the order never depends on how
// the notes were read. It returns a negative number when a goes first.
func CompareTasks(a, b *Header) int {
	if ra, rb := rank(a), rank(b); ra != rb {
		return ra - rb
	}
	da, _ := a.DueDate()
	db, _ := b.DueDate()
	if c := strings.Compare(da, db); c != 0 {
		return c
	}
	// Updated is YYYY-MM-DD, so it compares as text; one missing sorts first, as the
	// oldest.
	if c := strings.Compare(a.Updated, b.Updated); c != 0 {
		return c
	}
	return strings.Compare(a.ID, b.ID)
}
