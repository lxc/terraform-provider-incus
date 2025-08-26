package main

import (
	"strings"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Capital capitalizes the given string ("foo" -> "Foo").
func Capital(s string) string {
	return cases.Title(language.English, cases.NoLower).String(s)
}

var acronyms = map[string]struct{}{
	"acl":  {},
	"url":  {},
	"snat": {},
}

// CamelCase converts to camel case ("foo_bar" -> "fooBar").
// If a segment (with the exception of the first one) is a known acronym,
// it is returned in all upper case.
func CamelCase(s string) string {
	words := capitalizedWords(s)
	words[0] = strings.ToLower(words[0])
	return strings.Join(words, "")
}

// PascalCase converts to pascal case ("foo_bar" -> "FooBar").
// If a segment is a known acronym, it is returned in all upper case.
func PascalCase(s string) string {
	return strings.Join(capitalizedWords(s), "")
}

// KebabCase converts to kebab case ("foo_bar" -> "foo-bar").
func KebabCase(s string) string {
	return strings.ToLower(strings.Join(capitalizedWords(s), "-"))
}

// TitleCase converts to title case ("foo_bar" -> "Foo Bar").
// If a segment is a known acronym, it is returned in all upper case.
func TitleCase(s string) string {
	return strings.Join(capitalizedWords(s), " ")
}

// Words converts to space delimited words ("foo_bar" -> "foo bar").
// If a segment is a known acronym, it is returned in all upper case.
func Words(s string) string {
	words := capitalizedWords(s)
	for i, w := range words {
		runes := []rune(w)
		if len(runes) > 1 && unicode.IsUpper(runes[1]) {
			continue
		}

		words[i] = strings.ToLower(w)
	}

	return strings.Join(words, " ")
}

func capitalizedWords(s string) []string {
	words := strings.Split(s, "_")
	for i := range words {
		_, ok := acronyms[strings.ToLower(words[i])]
		if ok {
			words[i] = strings.ToUpper(words[i])
			continue
		}

		words[i] = Capital(words[i])
	}

	return words
}
