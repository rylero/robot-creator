package cmd

import (
	"fmt"
	"strings"
	"unicode"
)

var javaKeywords = map[string]bool{
	"abstract": true, "assert": true, "boolean": true, "break": true,
	"byte": true, "case": true, "catch": true, "char": true,
	"class": true, "const": true, "continue": true, "default": true,
	"do": true, "double": true, "else": true, "enum": true,
	"extends": true, "final": true, "finally": true, "float": true,
	"for": true, "goto": true, "if": true, "implements": true,
	"import": true, "instanceof": true, "int": true, "interface": true,
	"long": true, "native": true, "new": true, "package": true,
	"private": true, "protected": true, "public": true, "return": true,
	"short": true, "static": true, "strictfp": true, "super": true,
	"switch": true, "synchronized": true, "this": true, "throw": true,
	"throws": true, "transient": true, "try": true, "void": true,
	"volatile": true, "while": true,
	"true": true, "false": true, "null": true,
}

func validateSubsystemName(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("subsystem name cannot be empty")
	}
	runes := []rune(name)
	if !unicode.IsLetter(runes[0]) {
		return fmt.Errorf("subsystem name must start with a letter, got %q", string(runes[0]))
	}
	for _, r := range runes[1:] {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return fmt.Errorf("subsystem name %q contains invalid character %q (letters, digits, underscore only)", name, string(r))
		}
	}
	if javaKeywords[strings.ToLower(name)] {
		return fmt.Errorf("%q is a Java reserved word", name)
	}
	return nil
}
