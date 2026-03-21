package cliout

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/sys/unix"
)

// Options controls human vs machine output for CLI commands.
type Options struct {
	JSON  bool
	Wide  bool
	Color bool
}

// DetectTTY reports whether stdout is a terminal (unless NO_COLOR is set).
func DetectTTY() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	fd := int(os.Stdout.Fd())
	_, err := unix.IoctlGetWinsize(fd, unix.TIOCGWINSZ)
	return err == nil
}

// Print marshals v as indented JSON to w.
func Print(w io.Writer, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}

// Table renders fixed-width columns; cells are truncated to maxCellRunes when not Wide.
func Table(w io.Writer, headers []string, rows [][]string, wide bool, maxCellRunes int) {
	if maxCellRunes <= 0 {
		maxCellRunes = 48
	}
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = utf8.RuneCountInString(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i >= len(widths) {
				break
			}
			c := cell
			if !wide && utf8.RuneCountInString(c) > maxCellRunes {
				c = truncateRunes(c, maxCellRunes-1) + "…"
			}
			n := utf8.RuneCountInString(c)
			if n > widths[i] {
				widths[i] = n
			}
		}
	}
	writeRow(w, headers, widths)
	sep := make([]string, len(widths))
	for i, wn := range widths {
		sep[i] = strings.Repeat("-", wn)
	}
	writeRow(w, sep, widths)
	for _, row := range rows {
		padded := make([]string, len(headers))
		for i := range headers {
			if i < len(row) {
				padded[i] = row[i]
				if !wide && utf8.RuneCountInString(padded[i]) > maxCellRunes {
					padded[i] = truncateRunes(padded[i], maxCellRunes-1) + "…"
				}
			}
		}
		writeRow(w, padded, widths)
	}
}

func writeRow(w io.Writer, cells []string, widths []int) {
	var b strings.Builder
	for i, cell := range cells {
		if i > 0 {
			b.WriteString("  ")
		}
		pad := widths[i] - utf8.RuneCountInString(cell)
		if pad < 0 {
			pad = 0
		}
		b.WriteString(cell)
		b.WriteString(strings.Repeat(" ", pad))
	}
	fmt.Fprintln(w, b.String())
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}
