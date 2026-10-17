//go:build windows

package i18n

import "golang.org/x/sys/windows"

// systemLanguage is the user's first display language on Windows (es-ES, en-US), which
// has no locale variables; empty if Windows does not say.
func systemLanguage() string {
	langs, err := windows.GetUserPreferredUILanguages(windows.MUI_LANGUAGE_NAME)
	if err != nil || len(langs) == 0 {
		return ""
	}
	return langs[0]
}
