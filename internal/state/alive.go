package state

// Alive says whether a process with this number runs on this computer. A number used
// again by another process counts as alive: callers only ever leave things alone then.
func Alive(pid int) bool { return alive(pid) }
