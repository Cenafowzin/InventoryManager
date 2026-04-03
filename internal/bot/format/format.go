package format

import (
	"fmt"
	"strings"
)

func Num(n float64) string {
	if n == float64(int(n)) {
		return fmt.Sprintf("%d", int(n))
	}
	return fmt.Sprintf("%.2f", n)
}

// Block formata uma seção com título e linhas.
func Block(title string, lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return fmt.Sprintf("**%s**\n%s", title, strings.Join(lines, "\n"))
}

// Divider retorna um separador visual.
func Divider() string {
	return "─────────────────────"
}
