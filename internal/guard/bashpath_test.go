package guard

import "testing"

// Git Bash on Windows writes C:\\Users as /c/Users; elsewhere such a path is left alone.
func TestBashPath(t *testing.T) {
	for in, want := range map[string]string{
		"/c/Users/ana": "C:/Users/ana",
		"/c/":          "C:/",
		"/c":           "C:/",
		"/D/work":      "D:/work",
		"/usr/bin":     "/usr/bin",
		"/cygdrive/c":  "/cygdrive/c",
		"C:/Users":     "C:/Users",
		"relative":     "relative",
	} {
		if got := bashPath(in, "windows"); got != want {
			t.Errorf("windows %q: %q, want %q", in, got, want)
		}
		if got := bashPath(in, "linux"); got != in {
			t.Errorf("linux %q: %q, changed", in, got)
		}
	}
}
