//go:build !windows

package i18n

// systemLanguage is empty where the locale variables are the system's setting.
func systemLanguage() string { return "" }
