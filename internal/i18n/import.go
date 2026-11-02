package i18n

// `augmented-brain import`.
var importMessages = map[string]msg{
	"import.usage": {
		en: "usage: augmented-brain import <folder> [--map file] [--repos-base folder] [--apply]",
		es: "uso: augmented-brain import <carpeta> [--map fichero] [--repos-base carpeta] [--apply]",
	},
	"import.space": {
		en: "%s: %s",
		es: "%s: %s",
	},
	"import.named": {
		en: "space %s",
		es: "espacio %s",
	},
	"import.shared": {
		en: "shared notes (every space)",
		es: "notas compartidas (todos los espacios)",
	},
	"import.left": {
		en: "%d files are left out (see below).",
		es: "%d ficheros se quedan fuera (abajo, por qué).",
	},
	"import.dry": {
		en: "Nothing was written: this is what would be imported. To write it: augmented-brain %s",
		es: "No se ha escrito nada: esto es lo que se importaría. Para escribirlo: augmented-brain %s",
	},
	"import.done": {
		en: "✓ %[3]s: %[1]d notes written, and %[2]d files copied as they were (archive and documents).",
		es: "✓ %[3]s: %[1]d notas escritas, y %[2]d ficheros copiados tal cual (archivo y documentos).",
	},
	"import.inbox": {
		en: "✓ %[2]s: %[1]d captures added to the inbox.",
		es: "✓ %[2]s: %[1]d capturas añadidas al buzón.",
	},
	"import.kind.inbox": {
		en: "inbox captures",
		es: "capturas del buzón",
	},
	"import.kind.docs": {
		en: "task documents",
		es: "documentos de tareas",
	},
	"import.kind.task": {
		en: "tasks",
		es: "tareas",
	},
	"import.kind.area": {
		en: "areas",
		es: "áreas",
	},
	"import.kind.reference": {
		en: "references",
		es: "referencias",
	},
	"import.kind.lesson": {
		en: "lessons",
		es: "lecciones",
	},
	"import.kind.rule": {
		en: "rules",
		es: "reglas",
	},
	"import.kind.person": {
		en: "people",
		es: "personas",
	},
	"import.kind.archive": {
		en: "archive files",
		es: "ficheros de archivo",
	},
	"import.issue.not-a-note": {
		en: "Not notes, left out (%d): no header, such as an index.",
		es: "No son notas, se quedan fuera (%d): sin cabecera, como un índice.",
	},
	"import.issue.unreadable": {
		en: "Headers that cannot be read, left out (%d):",
		es: "Cabeceras que no se pueden leer, se quedan fuera (%d):",
	},
	"import.issue.to-profile": {
		en: "About you, for your profile (%d): copy what matters into it by hand.",
		es: "Sobre ti, para tu perfil (%d): copia a mano lo que importe.",
	},
	"import.issue.no-space": {
		en: "No space for their area, left out (%d): add the area to the map, or a line \"* = <space>\".",
		es: "Sin espacio para su área, se quedan fuera (%d): añade el área a la tabla, o una línea \"* = <espacio>\".",
	},
	"import.issue.not-shared": {
		en: "Sent to the shared notes, but only rules, references and lessons go there; left out (%d):",
		es: "Enviadas a las notas compartidas, pero ahí solo van reglas, referencias y lecciones; se quedan fuera (%d):",
	},
	"import.issue.secret": {
		en: "A secret written in them (%d): they are imported, but the backup refuses to upload them until it is taken out. Keep it in your keychain or a secrets manager:",
		es: "Llevan un secreto escrito (%d): se importan, pero la copia de seguridad no las sube hasta que se quite. Guárdalo en tu llavero o en un gestor de secretos:",
	},
	"import.issue.taken": {
		en: "Already a note of the space, left out (%d):",
		es: "Ya son notas del espacio, se quedan fuera (%d):",
	},
	"import.issue.duplicate": {
		en: "Two files with the same id, the second left out (%d):",
		es: "Dos ficheros con el mismo id, el segundo se queda fuera (%d):",
	},
	"import.issue.cross-link": {
		en: "Related to a note of another space or of none, link dropped (%d):",
		es: "Relacionadas con una nota de otro espacio o de ninguno, enlace quitado (%d):",
	},
	"import.issue.missing-area": {
		en: "Area not in their space, area dropped (%d):",
		es: "Área que no está en su espacio, área quitada (%d):",
	},
	"import.issue.dropped-field": {
		en: "Header fields augmented brain does not have, dropped (%d):",
		es: "Campos de cabecera que augmented brain no tiene, quitados (%d):",
	},
	"import.issue.task-defaults": {
		en: "Tasks with no status: parked, waiting on you, to review (%d):",
		es: "Tareas sin estado: aparcadas, esperándote, para revisar (%d):",
	},
	"import.issue.repo-outside": {
		en: "Repositories outside the space's work folders, kept as full paths (%d):",
		es: "Repositorios fuera de las carpetas de trabajo del espacio, con su ruta completa (%d):",
	},
	"import.issue.unknown-type": {
		en: "No type to go by, imported as references (%d):",
		es: "Sin tipo del que fiarse, importadas como referencias (%d):",
	},
	"import.issue.archive-no-space": {
		en: "Archive files that belong to no note and no space, left out (%d):",
		es: "Ficheros de archivo que no son de ninguna nota ni espacio, se quedan fuera (%d):",
	},
}
