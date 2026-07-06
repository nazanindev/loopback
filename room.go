package main

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/gliderlabs/ssh"
)

type LineKind int

const (
	Narrate LineKind = iota
	Speech
	Keeper
)

type Line struct {
	kind LineKind
	who  string
	text string
	only string // if set, only this patron sees it
}

type Patron struct {
	name       string
	session    ssh.Session
	w, h       int
	input      []rune
	drink      string // "a rye"
	drinkShort string // "rye"
	joined     time.Time
	said       int
	poured     int
	visits     int
	gone       bool
	writeMu    sync.Mutex
}

func (p *Patron) write(s string) {
	p.writeMu.Lock()
	defer p.writeMu.Unlock()
	p.session.Write([]byte(s))
}

type Room struct {
	mu             sync.Mutex
	patrons        []*Patron
	regulars       []*NPC
	lines          []Line
	book           *Guestbook
	frame          int
	keeperBusyTill time.Time
	lastCallDay    int64
}

func NewRoom(dataDir string) *Room {
	r := &Room{book: NewGuestbook(dataDir), regulars: theRegulars()}
	r.rosterCheck(true)
	r.addLine(Line{kind: Narrate, text: "the bar is open. it was open before you got here."})
	return r
}

// nightSeed identifies the bar-night: a date that doesn't roll over until 6am,
// because nobody who is here at 2am believes it is tomorrow yet.
func nightSeed() int64 {
	t := time.Now().Add(-6 * time.Hour)
	return int64(t.Year()*10000 + int(t.Month())*100 + t.Day())
}

func (r *Room) weather() string {
	rng := rand.New(rand.NewSource(nightSeed()))
	switch x := rng.Float64(); {
	case x < 0.45:
		return "rain"
	case x < 0.65:
		return "drizzle"
	case x < 0.90:
		return "clear"
	default:
		return "fog"
	}
}

func (r *Room) weatherWord() string {
	switch r.weather() {
	case "rain":
		return "rain on the glass"
	case "drizzle":
		return "drizzle"
	case "fog":
		return "fog out"
	default:
		return moonPhrase() + " out"
	}
}

func (r *Room) weatherClause() string {
	switch r.weather() {
	case "rain":
		return "the rain kept on."
	case "drizzle":
		return "the drizzle never quite committed."
	case "fog":
		return "the fog stayed."
	default:
		return "clear night, " + moonPhrase() + "."
	}
}

func (r *Room) entranceFlavor() string {
	switch r.weather() {
	case "rain":
		return ", shaking the rain off"
	case "drizzle":
		return ", collar up"
	case "fog":
		return " with the fog behind them"
	default:
		return ""
	}
}

func moonPhrase() string {
	const synodic = 29.530588853
	ref := time.Date(2000, 1, 6, 18, 14, 0, 0, time.UTC)
	age := math.Mod(time.Since(ref).Hours()/24, synodic)
	switch {
	case age < 1.5 || age > 28:
		return "a new moon"
	case age < 6.5:
		return "a young crescent moon"
	case age < 8.5:
		return "a first-quarter moon"
	case age < 13.5:
		return "a waxing moon"
	case age < 16.5:
		return "a full moon"
	case age < 21.5:
		return "a waning moon"
	case age < 23.5:
		return "a last-quarter moon"
	default:
		return "an old crescent moon"
	}
}

func (r *Room) addLine(l Line) {
	r.lines = append(r.lines, l)
	if len(r.lines) > 120 {
		r.lines = r.lines[len(r.lines)-120:]
	}
}

type outFrame struct {
	p    *Patron
	data string
}

func (r *Room) Tick() {
	t := time.NewTicker(400 * time.Millisecond)
	for range t.C {
		r.mu.Lock()
		r.frame++
		if r.frame%8 == 0 {
			r.ambient()
		}
		if r.frame%150 == 1 {
			r.rosterCheck(false)
		}
		r.checkLastCall()
		frames := make([]outFrame, 0, len(r.patrons))
		for _, p := range r.patrons {
			frames = append(frames, outFrame{p, r.compose(p)})
		}
		r.mu.Unlock()
		for _, f := range frames {
			f.p.write(f.data)
		}
	}
}

func (r *Room) checkLastCall() {
	now := time.Now()
	if now.Hour() == 1 && now.Minute() == 45 && r.lastCallDay != nightSeed() {
		r.lastCallDay = nightSeed()
		r.addLine(Line{kind: Keeper, text: "last call. no hurry."})
	}
}

func (r *Room) renderOne(p *Patron) {
	r.mu.Lock()
	if p.gone {
		r.mu.Unlock()
		return
	}
	f := r.compose(p)
	r.mu.Unlock()
	p.write(f)
}

func (r *Room) Seat(s ssh.Session) {
	pty, winCh, isPty := s.Pty()
	if !isPty {
		fmt.Fprintln(s, "the loopback needs a terminal. try again with:  ssh -t -p <port> you@<host>")
		s.Exit(1)
		return
	}
	name := cleanName(s.User())
	r.mu.Lock()
	for _, q := range r.patrons {
		if q.name == name {
			name = name + "-2"
		}
	}
	phrase, short := drinkFor(name)
	p := &Patron{
		name: name, session: s,
		w: pty.Window.Width, h: pty.Window.Height,
		drink: phrase, drinkShort: short,
		joined: time.Now(), visits: r.book.Visits(name),
	}
	if p.w <= 0 {
		p.w = 80
	}
	if p.h <= 0 {
		p.h = 24
	}
	r.patrons = append(r.patrons, p)
	r.addLine(Line{kind: Narrate, text: name + " comes in" + r.entranceFlavor() + "."})
	r.mu.Unlock()

	p.write("\x1b[?1049h\x1b[?25l\x1b[2J")
	r.renderOne(p)
	r.keeperGreets(p)

	go func() {
		for wc := range winCh {
			r.mu.Lock()
			p.w, p.h = wc.Width, wc.Height
			r.mu.Unlock()
			r.renderOne(p)
		}
	}()

	r.readInput(p)
	r.depart(p)
}

func (r *Room) depart(p *Patron) {
	r.mu.Lock()
	p.gone = true
	for i, q := range r.patrons {
		if q == p {
			r.patrons = append(r.patrons[:i], r.patrons[i+1:]...)
			break
		}
	}
	r.addLine(Line{kind: Narrate, text: p.name + " settles up and steps out."})
	clause := r.weatherClause()
	r.mu.Unlock()
	r.book.Record(p.name, p.drinkShort, clause, time.Since(p.joined), p.said)
	p.write("\x1b[?1049l\x1b[?25h")
	fmt.Fprintln(p.session, "the door swings shut behind you.")
	p.session.Exit(0)
}

func (r *Room) readInput(p *Patron) {
	buf := make([]byte, 256)
	esc := 0
	for {
		n, err := p.session.Read(buf)
		if err != nil {
			return
		}
		for _, b := range buf[:n] {
			switch {
			case esc == 1:
				if b == '[' || b == 'O' {
					esc = 2
				} else {
					esc = 0
				}
			case esc == 2:
				if (b >= 'A' && b <= 'Z') || b == '~' {
					esc = 0
				}
			case b == 0x1b:
				esc = 1
			case b == 3 || b == 4: // ctrl-c, ctrl-d
				return
			case b == '\r' || b == '\n':
				r.submit(p)
			case b == 0x7f || b == 8:
				r.mu.Lock()
				if len(p.input) > 0 {
					p.input = p.input[:len(p.input)-1]
				}
				r.mu.Unlock()
			default:
				if b >= 32 && b < 127 {
					r.mu.Lock()
					if len(p.input) < 120 {
						p.input = append(p.input, rune(b))
					}
					r.mu.Unlock()
				}
			}
		}
		r.renderOne(p)
	}
}

func (r *Room) submit(p *Patron) {
	r.mu.Lock()
	text := strings.TrimSpace(string(p.input))
	p.input = p.input[:0]
	if text == "" {
		r.mu.Unlock()
		return
	}
	if text == "/tab" {
		var t string
		switch n := p.poured; {
		case n == 0:
			t = "nothing on it yet. give it a minute."
		case n == 1:
			t = p.drinkShort + ", once. on the house. it's always on the house."
		default:
			t = fmt.Sprintf("%s, times %d. on the house. it's always on the house.", p.drinkShort, n)
		}
		r.addLine(Line{kind: Keeper, text: t, only: p.name})
		r.mu.Unlock()
		return
	}
	if len(text) > 90 {
		text = text[:90]
	}
	r.addLine(Line{kind: Speech, who: p.name, text: text})
	p.said++
	r.mu.Unlock()
	r.keeperReply(p, text)
}

func cleanName(u string) string {
	u = strings.ToLower(u)
	var sb strings.Builder
	for _, c := range u {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			sb.WriteRune(c)
		}
	}
	n := sb.String()
	if n == "" || n == "root" {
		n = "stranger"
	}
	if len(n) > 14 {
		n = n[:14]
	}
	return n
}
