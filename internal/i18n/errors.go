package i18n

import (
	"errors"
)

// Error is a message of the catalogue used as an error. Its Error() is the English
// text, the same one the program always gave, for the assistant and the logs; Message
// puts it in the language of the person who reads it, as the panel does.
type Error struct {
	Key  string
	Args []any
}

// Errorf returns the message key, with its arguments, as an error.
func Errorf(key string, args ...any) *Error { return &Error{Key: key, Args: args} }

func (e *Error) Error() string { return e.in("en") }

// Unwrap gives the errors among the arguments, so errors.Is and errors.As see them.
func (e *Error) Unwrap() []error {
	var out []error
	for _, a := range e.Args {
		if err, ok := a.(error); ok {
			out = append(out, err)
		}
	}
	return out
}

func (e *Error) in(lang string) string {
	args := make([]any, len(e.Args))
	for i, a := range e.Args {
		switch a := a.(type) {
		case Word:
			args[i] = T(lang, string(a))
		case error:
			args[i] = Message(lang, a)
		default:
			args[i] = a
		}
	}
	return T(lang, e.Key, args...)
}

// Word is an argument of an Error that is itself a message key (the name of a field,
// say): it is put in the same language as the message around it.
type Word string

// Message is err in lang: the message of the catalogue when err is one, or wraps one
// (what wraps it then adds nothing a person needs), and its own text otherwise.
func Message(lang string, err error) string {
	if err == nil {
		return ""
	}
	var e *Error
	if errors.As(err, &e) {
		return e.in(lang)
	}
	return err.Error()
}

// Errors of the notes: what brain says when a change cannot be made.
var errorMessages = map[string]msg{
	"err.toolong": {
		en: "nothing was saved: the %s is %d characters long, and the most is %d; %s",
		es: "no se ha guardado nada: %s tiene %d caracteres, y el máximo es %d; %s",
	},
	"err.toolong.oneline": {
		en: "keep it to one line here, and put the rest in the note's body or, for a task, in its journal (task_journal) or a file under docs/<task>/ that the journal names",
		es: "déjalo aquí en una línea, y pon el resto en el cuerpo de la nota o, en una tarea, en su diario o en un fichero de docs/<tarea>/ que el diario nombre",
	},
	"err.toolong.text": {
		en: "put the long text in a file under docs/<task>/ in the notes folder, and write here a line that names it",
		es: "pon el texto largo en un fichero de docs/<tarea>/ en la carpeta de notas, y escribe aquí una línea que lo nombre",
	},
	"err.toolong.name": {
		en: "a name goes here; the rest goes in about or in the personality",
		es: "aquí va un nombre; el resto va en «sobre mí» o en la personalidad",
	},
	"err.toolong.profile": {
		en: "keep in the profile who the user is and how they like to work, and put the rest in knowledge notes (knowledge_new)",
		es: "deja en el perfil quién es el usuario y cómo le gusta trabajar, y pon el resto en notas de conocimiento",
	},
	"err.what.title":     {en: "title", es: "el título"},
	"err.what.summary":   {en: "summary", es: "el resumen"},
	"err.what.next":      {en: "next step", es: "el siguiente paso"},
	"err.what.waiting":   {en: "waiting_on", es: "«espera a»"},
	"err.what.watch":     {en: "watch date's what", es: "lo que hay que mirar en la fecha"},
	"err.what.note":      {en: "note", es: "la nota"},
	"err.what.journal":   {en: "journal entry", es: "la entrada del diario"},
	"err.what.capture":   {en: "capture", es: "la captura"},
	"err.what.profile":   {en: "profile", es: "el perfil"},
	"err.what.username":  {en: "user's name", es: "tu nombre"},
	"err.what.assistant": {en: "assistant's name", es: "el nombre del asistente"},
	"err.secret.in": {
		en: "nothing was saved: the text has %s in it. Secrets never go into notes, and the backup would refuse the whole space: keep it in the system keychain or a secrets manager, and write down only where it is",
		es: "no se ha guardado nada: el texto lleva %s. Los secretos nunca van a las notas, y la copia de seguridad rechazaría todo el espacio: guárdalo en el llavero del sistema o en un gestor de secretos, y apunta solo dónde está",
	},
	"err.note.invalid": {
		en: "%s is not a valid note, so it is not written over; fix it first (`augmented-brain check` says what is wrong)",
		es: "%s no es una nota válida, así que no se escribe encima; arréglala antes (`augmented-brain check` dice qué está mal)",
	},
	"err.note.missing": {
		en: "there is no %s %q",
		es: "no hay ninguna nota %s con id %q",
	},
	"err.note.kind": {
		en: "%q is a %s, not a %s",
		es: "%q es de tipo %s, no %s",
	},
	"err.task.title": {
		en: "a task needs a title",
		es: "una tarea necesita un título",
	},
	"err.task.parent": {
		en: "the parent must be an existing task: %v",
		es: "la tarea madre tiene que existir: %v",
	},
	"err.task.deep": {
		en: "%q is already a subtask of %q, and subtasks go one level deep only: make this one a subtask of %q, or a step in the Plan of %q",
		es: "%q ya es subtarea de %q, y las subtareas solo tienen un nivel: haz esta subtarea de %q, o un paso del Plan de %q",
	},
	"err.task.parentclosed": {
		en: "%q is closed: a closed task takes no new subtasks",
		es: "%q está cerrada: una tarea cerrada no admite subtareas nuevas",
	},
	"err.task.openchildren": {
		en: "%q cannot be closed while its subtasks are open; close them first: %s",
		es: "%q no se puede cerrar con subtareas abiertas; ciérralas antes: %s",
	},
	"err.task.clear": {
		en: "%q cannot be cleared",
		es: "%q no se puede vaciar",
	},
	"err.watch.what": {
		en: "a watch date needs what to look at",
		es: "una fecha para mirar algo necesita qué hay que mirar",
	},
	"err.watch.none": {
		en: "task %s has no open watch date on %s",
		es: "la tarea %s no tiene ninguna fecha abierta el %s",
	},
	"err.date": {
		en: "date %q: use YYYY-MM-DD",
		es: "fecha %q: escríbela como AAAA-MM-DD",
	},
	"err.journal.empty": {
		en: "the journal entry is empty",
		es: "la entrada del diario está vacía",
	},
	"err.park.next": {
		en: "parking needs the next step: exactly how to resume",
		es: "para aparcar hace falta el siguiente paso: cómo retomarla exactamente",
	},
	"err.task.closed": {
		en: "task %s is closed: to open it again, say why with reopen",
		es: "la tarea %s está cerrada: para abrirla otra vez, di por qué al reabrirla",
	},
	"err.close.result": {
		en: "closing needs the result",
		es: "para cerrar hace falta el resultado",
	},
	"err.close.already": {
		en: "task %s is already closed: its result stays; to add to it, write in its journal",
		es: "la tarea %s ya está cerrada: su resultado se queda; para añadir algo, escríbelo en su diario",
	},
	"err.space.noconfig": {
		en: "no configuration to find other spaces in",
		es: "no hay configuración en la que buscar otros espacios",
	},
	"err.space.missing": {
		en: "there is no space %q; the spaces are: %s",
		es: "no hay ningún espacio %q; los espacios son: %s",
	},
	"err.capture.empty": {
		en: "nothing to capture",
		es: "no hay nada que capturar",
	},
	"err.area.title": {
		en: "an area needs a title",
		es: "un área necesita un título",
	},
	"err.knowledge.kind": {
		en: "kind %q: use reference or lesson",
		es: "tipo %q: usa reference o lesson",
	},
	"err.note.title": {
		en: "a note needs a title",
		es: "una nota necesita un título",
	},
	"err.rule.body": {
		en: "a rule needs a title and a body: what to do, and why",
		es: "una regla necesita un título y un cuerpo: qué hacer, y por qué",
	},
	"err.note.exists": {
		en: "%s already exists (%q): nothing was created. If it is the same thing, %s; if it is another one, use another title",
		es: "%s ya existe (%q): no se ha creado nada. Si es lo mismo, %s; si es otra cosa, usa otro título",
	},
	"err.note.exists.call": {
		en: "call again with id %s to change it",
		es: "vuelve a llamar con id %s para cambiarla",
	},
	"err.note.exists.area": {
		en: "change it in the panel's Areas page",
		es: "cámbiala en la página Áreas del panel",
	},
	"err.rule.area": {
		en: "a rule has no area",
		es: "una regla no tiene área",
	},
	"err.retire.why": {
		en: "say why it is retired: the reason stays with it",
		es: "di por qué se retira: el motivo se queda con ella",
	},
	"err.retire.self": {
		en: "a note cannot replace itself",
		es: "una nota no puede sustituirse a sí misma",
	},
	"err.retire.missing": {
		en: "there is no note %q to replace it",
		es: "no hay ninguna nota %q que la sustituya",
	},
	"err.note.unchangeable": {
		en: "%q is a %s: only rules, references and lessons are changed this way",
		es: "%q es de tipo %s: así solo se cambian reglas, referencias y lecciones",
	},
	"err.repo.which": {
		en: "say which repository: its path, or its name as a task's repos: writes it",
		es: "di qué repositorio: su ruta, o su nombre tal como lo escribe el campo repos: de una tarea",
	},
	"err.repo.notfolder": {
		en: "%q is not a folder in this space's work folders on this computer",
		es: "%q no es una carpeta de las carpetas de trabajo de este espacio en este ordenador",
	},
	"err.repo.outside": {
		en: "%s is outside this space's work folders (%s): only repositories inside them are described here",
		es: "%s está fuera de las carpetas de trabajo de este espacio (%s): aquí solo se describen los repositorios de dentro",
	},
	"err.repo.path": {
		en: "%q cannot be written in the note: its path has a backquote or a line break",
		es: "%q no se puede escribir en la nota: su ruta lleva una comilla invertida o un salto de línea",
	},
	"err.repo.long": {
		en: "the description is %d characters long; keep it to one line of %d at most",
		es: "la descripción tiene %d caracteres; déjala en una línea de %d como mucho",
	},
	"err.repo.note": {
		en: "%s is a %s, not the reference of the repositories; rename it first",
		es: "%s es de tipo %s, no la referencia de los repositorios; cámbiale el nombre antes",
	},
	"err.profile.name": {
		en: "the profile needs the user's name",
		es: "el perfil necesita tu nombre",
	},
	"err.step.none": {
		en: "this task has no steps: they are the checkboxes (- [ ] text) of its Plan",
		es: "esta tarea no tiene pasos: son las casillas (- [ ] texto) de su Plan",
	},
	"err.step.number": {
		en: "there is no step %d: the plan has %d",
		es: "no hay paso %d: el plan tiene %d",
	},
	"err.step.which": {
		en: "say which step: its number or its text",
		es: "di qué paso: su número o su texto",
	},
	"err.step.notext": {
		en: "no step says %q",
		es: "ningún paso dice %q",
	},
	"err.step.several": {
		en: "%q matches several steps: %s; give its number",
		es: "%q coincide con varios pasos: %s; da su número",
	},
	"err.step.empty": {
		en: "the step is empty",
		es: "el paso está vacío",
	},
	"err.review.summary": {
		en: "a review needs a summary: what was decided. Nothing was saved, and the inbox is as it was",
		es: "una revisión necesita un resumen: qué se decidió. No se ha guardado nada, y el buzón está como estaba",
	},
	"err.review.inbox": {
		en: "%v\n(the inbox is as it was: no capture was taken out)",
		es: "%v\n(el buzón está como estaba: no se ha sacado ninguna captura)",
	},
	"err.person.name": {
		en: "a person needs a name",
		es: "una persona necesita un nombre",
	},
	"err.person.none": {
		en: "no such person in any space",
		es: "no hay esa persona en ningún espacio",
	},
	"err.person.nospace": {
		en: "a detail needs at least one space where it is visible",
		es: "un dato necesita al menos un espacio donde se vea",
	},
	"err.person.space": {
		en: "space %q is not one of the spaces of this machine",
		es: "el espacio %q no es ninguno de los espacios de esta máquina",
	},
	"err.person.notperson": {
		en: "%s in %s is not a person",
		es: "%s en %s no es una persona",
	},
	"err.person.spaces": {
		en: "%q is not one of the spaces: %s",
		es: "%q no es ninguno de los espacios: %s",
	},
	"err.person.nowhere": {
		en: "no space to save the person in",
		es: "no hay ningún espacio donde guardar a la persona",
	},
	"err.session.none": {
		en: "no such session to recover",
		es: "no hay esa sesión que recuperar",
	},
	"err.secret.name": {
		en: "service and name are single words (letters, digits, . _ @ + -, not starting with -), and what it is for one line of plain text, up to 160 characters",
		es: "servicio y nombre son una sola palabra (letras, cifras, . _ @ + -, sin empezar por -), y para qué sirve, una línea de texto normal de hasta 160 caracteres",
	},
	"err.secret.value": {
		en: "that looks like a secret's value: only its name and what it is for are kept",
		es: "eso parece el valor de un secreto: solo se guardan su nombre y para qué sirve",
	},
	"err.secret.missing": {
		en: "there is no secret %s in the list",
		es: "no hay ningún secreto %s en la lista",
	},
}
