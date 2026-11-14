package i18n

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestError(t *testing.T) {
	inner := Errorf("err.task.title")
	e := Errorf("err.task.parent", inner)
	if got := e.Error(); got != "the parent must be an existing task: a task needs a title" {
		t.Errorf("Error() = %q", got)
	}
	if got := Message("es", e); got != "la tarea madre tiene que existir: una tarea necesita un título" {
		t.Errorf("Message(es) = %q", got)
	}
	if !errors.Is(e, inner) {
		t.Error("errors.Is does not see the error among the arguments")
	}
	// Wrapped by something that adds nothing a person needs: still in the reader's language.
	if got := Message("es", fmt.Errorf("x: %w", Errorf("err.date", "mañana"))); got != `fecha "mañana": escríbela como AAAA-MM-DD` {
		t.Errorf("wrapped = %q", got)
	}
	w := Errorf("err.toolong", Word("err.what.title"), 300, 200, Word("err.toolong.oneline"))
	if got := Message("es", w); !strings.HasPrefix(got, "no se ha guardado nada: el título tiene 300 caracteres") {
		t.Errorf("words = %q", got)
	}
	if Message("es", errors.New("plain")) != "plain" || Message("es", nil) != "" {
		t.Error("a plain error keeps its text")
	}
}
