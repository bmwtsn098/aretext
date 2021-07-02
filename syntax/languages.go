// This file is generated by gen_tokenizers.go.  DO NOT EDIT.
package syntax

import (
	"log"

	"github.com/aretext/aretext/syntax/parser"
)

// Language is an enum of available languages that we can parse.
type Language int

const (
	LanguagePlaintext = Language(iota)
	LanguageJson
	LanguageYaml
	LanguageGo
	LanguageGitCommit
	LanguageGitRebase
	LanguageDevlog
)

var AllLanguages = []Language{
	LanguageJson,
	LanguageYaml,
	LanguageGo,
	LanguageGitCommit,
	LanguageGitRebase,
	LanguageDevlog,
}

func (language Language) String() string {
	switch language {
	case LanguagePlaintext:
		return "plaintext"
	case LanguageJson:
		return "json"
	case LanguageYaml:
		return "yaml"
	case LanguageGo:
		return "go"
	case LanguageGitCommit:
		return "gitcommit"
	case LanguageGitRebase:
		return "gitrebase"
	case LanguageDevlog:
		return "devlog"
	default:
		return ""
	}
}

func LanguageFromString(s string) Language {
	switch s {
	case "plaintext":
		return LanguagePlaintext
	case "json":
		return LanguageJson
	case "yaml":
		return LanguageYaml
	case "go":
		return LanguageGo
	case "gitcommit":
		return LanguageGitCommit
	case "gitrebase":
		return LanguageGitRebase
	case "devlog":
		return LanguageDevlog
	default:
		log.Printf("Unrecognized syntax language '%s'\n", s)
		return LanguagePlaintext
	}
}

// TokenizerForLanguage returns a tokenizer for the specified language.
// If no tokenizer is available (e.g. for LanguagePlaintext) this returns nil.
func TokenizerForLanguage(language Language) *parser.Tokenizer {
	switch language {
	case LanguageJson:
		return JsonTokenizer
	case LanguageYaml:
		return YamlTokenizer
	case LanguageGo:
		return GoTokenizer
	case LanguageGitCommit:
		return GitCommitTokenizer
	case LanguageGitRebase:
		return GitRebaseTokenizer
	case LanguageDevlog:
		return DevlogTokenizer
	default:
		return nil
	}
}
