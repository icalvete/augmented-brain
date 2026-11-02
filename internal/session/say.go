package session

// The hooks and the MCP tools speak to the assistant, never to the user: it must pass
// their news on in the user's language, which only it can tell from the conversation.
// These pieces keep that request worded the same way everywhere.

// Language names the configuration's language, the one the assistant falls back on
// while it cannot tell the user's.
func Language(lang string) string {
	if lang == "es" {
		return "Spanish"
	}
	return "English"
}

// InUserLanguage asks the assistant to write in the user's language.
func InUserLanguage(lang string) string {
	return "in the language the user writes in (" + Language(lang) + " if you cannot tell)"
}

// InTheirLanguage is InUserLanguage for a sentence that has just named the user.
func InTheirLanguage(lang string) string {
	return "in the language they write in (" + Language(lang) + " if you cannot tell)"
}

// FirstAnswerLine starts a request to pass something on, briefly, in the first answer;
// what to say follows it.
func FirstAnswerLine(lang string) string {
	return "In one short line of your first answer, " + InUserLanguage(lang) + ", "
}

// TellUserLine asks the assistant to tell the user, briefly, in its first answer.
func TellUserLine(lang string) string {
	return "Tell the user in one short line of your first answer, " + InTheirLanguage(lang)
}
