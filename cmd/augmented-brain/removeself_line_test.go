package main

import (
	"errors"
	"strings"
	"testing"
)

// The cleanup line names every program by its full path, so nothing is looked for in
// the current folder, and refuses a path cmd.exe would change (R-15).
func TestRemoveSelfLine(t *testing.T) {
	exe, line, err := removeSelfLine(`C:\Users\lucia\AppData\Local\augmented-brain\augmented-brain.exe`, `C:\Windows`)
	if err != nil {
		t.Fatal(err)
	}
	if exe != `C:\Windows\System32\cmd.exe` {
		t.Errorf("exe = %s", exe)
	}
	want := `"C:\Windows\System32\cmd.exe" /d /s /c ""C:\Windows\System32\PING.EXE" -n 3 127.0.0.1 >nul & ` +
		`del /f /q "C:\Users\lucia\AppData\Local\augmented-brain\augmented-brain.exe" "C:\Users\lucia\AppData\Local\augmented-brain\augmented-brain.exe.old""`
	if line != want {
		t.Errorf("line:\n%s\nwant:\n%s", line, want)
	}
	if strings.Contains(line, " ping ") {
		t.Error("ping without its path")
	}
	for _, self := range []string{`C:\Users\a%b%c\augmented-brain.exe`, `C:\Users\a"b\augmented-brain.exe`, "C:\\a\nb\\augmented-brain.exe"} {
		if _, _, err := removeSelfLine(self, `C:\Windows`); !errors.Is(err, errSelfPath) {
			t.Errorf("%q: err = %v", self, err)
		}
	}
	// No SystemRoot, or one cmd.exe would change: the usual one.
	for _, root := range []string{"", `C:\%x%`} {
		if exe, _, _ := removeSelfLine(`C:\p\augmented-brain.exe`, root); exe != `C:\Windows\System32\cmd.exe` {
			t.Errorf("root %q: exe = %s", root, exe)
		}
	}
}
