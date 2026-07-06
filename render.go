package main

import (
	"strings"
	"time"
	"unicode/utf8"
)

const (
	cReset = "\x1b[0m"
	cTitle = "\x1b[1;38;5;222m"
	cAmber = "\x1b[38;5;179m"
	cInk   = "\x1b[38;5;251m"
	cName  = "\x1b[38;5;110m"
	cDim   = "\x1b[38;5;242m"
	cFaint = "\x1b[38;5;238m"
	cRain  = "\x1b[38;5;24m"
)

func vislen(s string) int { return utf8.RuneCountInString(s) }

func clip(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if vislen(s) <= n {
		return s
	}
	rs := []rune(s)
	if n == 1 {
		return "…"
	}
	return string(rs[:n-1]) + "…"
}

// compose draws one patron's view of the room. Callers hold r.mu.
func (r *Room) compose(p *Patron) string {
	w, h := p.w, p.h
	if w < 44 || h < 12 {
		return "\x1b[H\x1b[2J\r\n  the loopback.\r\n  the door is too narrow — make the terminal bigger.\r\n"
	}

	var b strings.Builder
	b.WriteString("\x1b[H")
	used := 0
	line := func(s string) {
		b.WriteString(s)
		b.WriteString("\x1b[K\r\n")
		used++
	}

	// the sign over the door
	title := "THE LOOPBACK"
	right := time.Now().Format("15:04") + " · " + r.weatherWord()
	pad := w - 4 - vislen(title) - vislen(right)
	if pad < 1 {
		pad = 1
	}
	line("")
	line("  " + cTitle + title + cReset + strings.Repeat(" ", pad) + cDim + right + cReset)
	line("")

	// the window and the shelf
	if w >= 66 && h >= 20 {
		win := r.windowArt()
		shelf := []string{
			cDim + "the shelf, bottles nobody orders from:" + cReset,
			cFaint + "OLD 127 · KEEPSAKE · KINDLING · PROTOCOL · TRUE NORTH" + cReset,
			"",
			cDim + "the bartender is behind the counter." + cReset,
			"",
			"",
		}
		for i, wl := range win {
			s := "  " + cRain + wl + cReset + "   "
			if i < len(shelf) {
				s += shelf[i]
			}
			line(s)
		}
		line("")
	}

	// the counter and whoever is at it
	line("  " + cFaint + strings.Repeat("═", w-4) + cReset)
	var seatRow strings.Builder
	seatRow.WriteString("  ")
	seen := 0
	for _, q := range r.patrons {
		c := cName
		if q == p {
			c = cAmber
		}
		if seen*16 > w-20 {
			seatRow.WriteString(cDim + "…others" + cReset)
			seen++
			break
		}
		seatRow.WriteString(c + "● " + q.name + cReset + "    ")
		seen++
	}
	for _, n := range r.regulars {
		if !n.present {
			continue
		}
		if seen*16 > w-20 {
			break
		}
		seatRow.WriteString(cDim + "● " + n.name + cReset + "    ")
		seen++
	}
	line(seatRow.String())
	line("")

	// the room itself: what has been said and done, most recent last
	logN := h - used - 3
	if logN < 3 {
		logN = 3
	}
	visible := make([]Line, 0, logN)
	for _, l := range r.lines {
		if l.only != "" && l.only != p.name {
			continue
		}
		visible = append(visible, l)
	}
	if len(visible) > logN {
		visible = visible[len(visible)-logN:]
	}
	for _, l := range visible {
		switch l.kind {
		case Narrate:
			line("  " + cDim + clip(l.text, w-4) + cReset)
		case Speech:
			line("  " + cName + l.who + cReset + cInk + ": " + clip(l.text, w-6-vislen(l.who)) + cReset)
		case Keeper:
			line("  " + cAmber + "the bartender: " + clip(l.text, w-19) + cReset)
		}
	}
	for used < h-3 {
		line("")
	}

	b.WriteString("  " + cTitle + "> " + cReset + cInk + string(p.input) + cAmber + "▌" + cReset + "\x1b[K\r\n")
	b.WriteString("  " + cFaint + "type to talk · /tab · ctrl-d leaves" + cReset + "\x1b[K")
	b.WriteString("\x1b[J")
	return b.String()
}

// windowArt draws six rows of whatever tonight is doing outside.
func (r *Room) windowArt() []string {
	const iw, ih = 22, 4
	weather := r.weather()
	rows := make([]string, 0, ih+2)
	rows = append(rows, "┌"+strings.Repeat("─", iw)+"┐")
	hsh := func(x, y int) int {
		v := uint32(x*374761393 + y*668265263 + 12345)
		v = (v ^ (v >> 13)) * 1274126177
		return int((v ^ (v >> 16)) % 100)
	}
	f := r.frame / 2
	for y := 0; y < ih; y++ {
		var sb strings.Builder
		sb.WriteString("│")
		for x := 0; x < iw; x++ {
			c := " "
			switch weather {
			case "rain":
				n := hsh(x, (y+f)%53)
				if n < 16 {
					c = "|"
				} else if n < 26 {
					c = "."
				}
			case "drizzle":
				if hsh(x, (y+f)%53) < 8 {
					c = "."
				}
			case "fog":
				if hsh(x, y+f/6) < 40 {
					c = "░"
				}
			case "clear":
				if hsh(x, y) < 4 {
					c = "·"
				}
				if x == 17 && y == 1 {
					c = "O"
				}
			}
			sb.WriteString(c)
		}
		sb.WriteString("│")
		rows = append(rows, sb.String())
	}
	rows = append(rows, "└"+strings.Repeat("─", iw)+"┘")
	return rows
}
