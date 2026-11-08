package i18n

// The history of the configuration: `augmented-brain config history | restore`, the
// panel's Versions page and its line in `doctor`.
var historyMessages = map[string]msg{
	"usage.config": {
		en: "usage: augmented-brain config history [N] | config restore N",
		es: "uso: augmented-brain config history [N] | config restore N",
	},
	"history.none": {
		en: "There are no versions of the configuration yet: one is kept the first time it changes.",
		es: "Aún no hay versiones de la configuración: se guarda una la primera vez que cambia.",
	},
	"history.head": {
		en: "Versions of your configuration, newest first (the last %d are kept, only on this computer):",
		es: "Versiones de tu configuración, la más reciente primero (se guardan las %d últimas, solo en este ordenador):",
	},
	"history.current": {
		en: "current",
		es: "la actual",
	},
	"history.changedsince": {
		en: "The configuration changed after the newest version (by hand, or by an earlier program); the next change of augmented brain keeps it as a version.",
		es: "La configuración ha cambiado después de la versión más reciente (a mano, o con un programa anterior); el próximo cambio de augmented brain la guarda como versión.",
	},
	"history.tip": {
		en: "To see what one changed: augmented-brain config history N. To go back to one: augmented-brain config restore N, or the panel's Versions page.",
		es: "Para ver qué cambió en una: augmented-brain config history N. Para volver a una: augmented-brain config restore N, o la página Versiones del panel.",
	},
	"history.badnumber": {
		en: "there is no version %s: see the list with augmented-brain config history",
		es: "no hay ninguna versión %s: mira la lista con augmented-brain config history",
	},
	"history.what": {
		en: "Version %d, %s, %s. What it changed from the one before:",
		es: "Versión %d, %s, %s. Lo que cambió respecto a la anterior:",
	},
	"history.first": {
		en: "Version %d, %s, %s: the oldest kept, so there is nothing to compare it with.",
		es: "Versión %d, %s, %s: la más antigua que se guarda, así que no hay con qué compararla.",
	},
	"history.same": {
		en: "It is the same as the current configuration: nothing to do.",
		es: "Es igual que la configuración actual: no hay nada que hacer.",
	},
	"history.restoring": {
		en: "Going back to version %d (%s). What changes:",
		es: "Vuelvo a la versión %d (%s). Lo que cambia:",
	},
	"history.restored": {
		en: "Done. The configuration you had is kept as a version, so this can be undone the same way.",
		es: "Hecho. La configuración que tenías queda guardada como una versión, así que esto se puede deshacer igual.",
	},
	"history.guardasks": {
		en: "The guard's lists changed: until you accept them in the panel's Guard page, or start a new session, the guard asks before every commit, push or merge it used to allow.",
		es: "Han cambiado las listas del guardia: hasta que las aceptes en la página Guardia del panel, o empieces una sesión nueva, el guardia pregunta antes de cada commit, push o mezcla que antes dejaba pasar.",
	},
	"history.badresult": {
		en: "that version cannot be used (%v); nothing changed",
		es: "esa versión no se puede usar (%v); no ha cambiado nada",
	},
	"history.filegone": {
		en: "(the file is removed)",
		es: "(el fichero se quita)",
	},
	"history.filenew": {
		en: "(the file is created)",
		es: "(el fichero se crea)",
	},
	"doctor.history.ok": {
		en: "configuration history: %d versions kept on this computer, the newest of %s",
		es: "historial de la configuración: %d versiones guardadas en este ordenador, la más reciente del %s",
	},
	"doctor.history.changed": {
		en: "the configuration changed after its newest version (%s), by hand or with an earlier program",
		es: "la configuración ha cambiado después de su versión más reciente (%s), a mano o con un programa anterior",
	},
	"doctor.history.changed.fix": {
		en: "nothing to do: it is kept as a version with the next change or session. To see the versions: augmented-brain config history, or the panel's Versions page",
		es: "no hay que hacer nada: se guarda como versión con el próximo cambio o sesión. Para ver las versiones: augmented-brain config history, o la página Versiones del panel",
	},
	// Who changed it.
	"history.by.outside": {
		en: "by hand or by another program",
		es: "a mano o con otro programa",
	},
	"history.by.command": {
		en: "command %s",
		es: "comando %s",
	},
	"history.by.tool": {
		en: "assistant (%s)",
		es: "asistente (%s)",
	},
	"history.by.panel": {
		en: "panel, page %s",
		es: "panel, página %s",
	},
	"history.by.panelany": {
		en: "panel",
		es: "panel",
	},
	"history.by.first": {
		en: "as it was when the history began",
		es: "tal como estaba al empezar el historial",
	},
	"history.by.restore": {
		en: "going back to an earlier version",
		es: "vuelta a una versión anterior",
	},
}
