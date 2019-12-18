package cli

import (
	"strings"

	wordwrap "github.com/mitchellh/go-wordwrap"
)

func WordWrap(lineWidth uint, texts ...string) string {
	text := strings.Join(texts, "")
	return wordwrap.WrapString(text, lineWidth)
}
