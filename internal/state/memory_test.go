package state

import "testing"

// Marks are per host and step, never go down, and the hosts that stopped are forgotten
// when a new mark is written.
func TestMemoryMarks(t *testing.T) {
	dir := t.TempDir()
	if got := MemoryWarned(dir, "100:10"); got != 0 {
		t.Errorf("no file: %d", got)
	}
	all := func(string) bool { return true }
	if err := MarkMemory(dir, "100:10", 2048, all); err != nil {
		t.Fatal(err)
	}
	if err := MarkMemory(dir, "200:20", 4096, all); err != nil {
		t.Fatal(err)
	}
	if got := MemoryWarned(dir, "100:10"); got != 2048 {
		t.Errorf("100: %d", got)
	}
	if got := MemoryWarned(dir, "100:11"); got != 0 {
		t.Errorf("a reused pid inherits the mark: %d", got)
	}
	if err := MarkMemory(dir, "200:20", 2048, all); err != nil {
		t.Fatal(err)
	}
	if got := MemoryWarned(dir, "200:20"); got != 4096 {
		t.Errorf("mark went down: %d", got)
	}
	// 100 ended: gone with the next write.
	if err := MarkMemory(dir, "300:30", 2048, func(k string) bool { return k != "100:10" }); err != nil {
		t.Fatal(err)
	}
	if MemoryWarned(dir, "100:10") != 0 || MemoryWarned(dir, "200:20") != 4096 || MemoryWarned(dir, "300:30") != 2048 {
		t.Errorf("after pruning: %+v", readMemory(dir))
	}
}
