package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Visit struct {
	Count    int       `json:"count"`
	Drink    string    `json:"drink"`
	LastSeen time.Time `json:"last_seen"`
	Minutes  int       `json:"minutes"`
}

type Guestbook struct {
	mu     sync.Mutex
	dir    string
	visits map[string]*Visit
}

func NewGuestbook(dir string) *Guestbook {
	g := &Guestbook{dir: dir, visits: map[string]*Visit{}}
	if b, err := os.ReadFile(g.jsonPath()); err == nil {
		json.Unmarshal(b, &g.visits)
	}
	return g
}

func (g *Guestbook) jsonPath() string { return filepath.Join(g.dir, "visits.json") }

func (g *Guestbook) Visits(name string) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	if v, ok := g.visits[name]; ok {
		return v.Count
	}
	return 0
}

func (g *Guestbook) Record(name, drink, weatherClause string, stayed time.Duration, said int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	v := g.visits[name]
	if v == nil {
		v = &Visit{}
		g.visits[name] = v
	}
	v.Count++
	v.Drink = drink
	v.LastSeen = time.Now()
	v.Minutes += int(stayed.Minutes())
	if b, err := json.MarshalIndent(g.visits, "", "  "); err == nil {
		os.WriteFile(g.jsonPath(), b, 0o644)
	}

	stayPhrase := "a few minutes"
	if m := int(stayed.Minutes()); m >= 5 {
		stayPhrase = fmt.Sprintf("%d minutes", m)
	}
	saidPhrase := "said nothing"
	switch {
	case said == 1:
		saidPhrase = "said one thing"
	case said > 1:
		saidPhrase = fmt.Sprintf("said %d things", said)
	}
	stamp := strings.ToLower(time.Now().Format("Jan 2, 3:04pm"))
	entry := fmt.Sprintf("%s — %s stayed %s, %s, drank %s. %s\n",
		stamp, name, stayPhrase, saidPhrase, drink, weatherClause)
	if f, err := os.OpenFile(filepath.Join(g.dir, "guestbook.txt"),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
		f.WriteString(entry)
		f.Close()
	}
}
