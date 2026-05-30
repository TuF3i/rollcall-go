package logger

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

var (
	sectionC = color.New(color.FgCyan, color.Bold)
	keyC     = color.New(color.FgYellow)
	valC     = color.New(color.FgGreen, color.Bold)
	dimC     = color.New(color.FgHiBlack)
	accentC  = color.New(color.FgWhite, color.Bold)
	tagOkC   = color.New(color.FgWhite, color.BgGreen, color.Bold)
	tagFailC = color.New(color.FgWhite, color.BgRed, color.Bold)
	tagWarnC = color.New(color.FgWhite, color.BgYellow, color.Bold)
)

// Section returns a colored section title string for use in slog messages.
func Section(title string) string {
	return sectionC.Sprint(title)
}

// K returns a colored key string.
func K(key string) string {
	return dimC.Sprint(" " + key)
}

// V returns a colored value string.
func V(val interface{}) string {
	return accentC.Sprint(fmt.Sprintf("%v", val))
}

// KV returns a "key=value" colored pair.
func KV(key string, val interface{}) string {
	return dimC.Sprint(" "+key+"=") + accentC.Sprint(fmt.Sprintf("%v", val))
}

// Dim returns a dimmed string.
func Dim(s string) string {
	return dimC.Sprint(s)
}

// TagOK returns a green-background success tag.
func TagOK(label string) string {
	return " " + tagOkC.Sprint(" "+label+" ")
}

// TagFail returns a red-background failure tag.
func TagFail(label string) string {
	return " " + tagFailC.Sprint(" "+label+" ")
}

// TagWarn returns a yellow-background warning tag.
func TagWarn(label string) string {
	return " " + tagWarnC.Sprint(" "+label+" ")
}

// Header returns a formatted header block for structured info display.
// Example output: "━━━ 登录成功 ━━━"
func Header(title string) string {
	line := strings.Repeat("━", 3)
	return dimC.Sprint(line) + sectionC.Sprint(title) + dimC.Sprint(line)
}
