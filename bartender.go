package main

import (
	"hash/fnv"
	"math/rand"
	"strings"
	"time"
)

var keeperIdle = []string{
	"the bartender polishes a glass that was already clean.",
	"the bartender holds a glass up to the light, satisfied about something.",
	"the bartender writes a line in a small book and puts it away.",
	"the bartender glances at the window and decides against comment.",
	"the bartender turns the radio down, though nobody heard it playing.",
	"the bartender straightens a bottle nobody has ever ordered from.",
}

var drinks = []struct{ phrase, short string }{
	{"a rye", "rye"},
	{"a stout", "stout"},
	{"a black coffee", "black coffee"},
	{"a gin, neat", "gin"},
	{"a soda water with a lime nobody asked for", "soda water"},
	{"a glass of the house red", "the house red"},
	{"hot water with lemon", "hot water with lemon"},
}

// drinkFor is stable: the bar decides what you drink the first time you walk
// in, and that is that.
func drinkFor(name string) (phrase, short string) {
	h := fnv.New32a()
	h.Write([]byte(name))
	d := drinks[h.Sum32()%uint32(len(drinks))]
	return d.phrase, d.short
}

func (r *Room) keeperGreets(p *Patron) {
	time.AfterFunc(2*time.Second, func() {
		r.mu.Lock()
		if p.gone {
			r.mu.Unlock()
			return
		}
		greeting := "what'll it be?"
		if p.visits > 0 {
			greeting = "evening, " + p.name + ". the usual?"
		}
		r.addLine(Line{kind: Keeper, text: greeting})
		r.keeperBusyTill = time.Now().Add(10 * time.Second)
		r.mu.Unlock()

		delay := 2500 * time.Millisecond
		suffix := "."
		if p.visits == 0 {
			delay = 4 * time.Second
			suffix = ", without waiting for an answer."
		}
		time.AfterFunc(delay, func() {
			r.mu.Lock()
			defer r.mu.Unlock()
			if p.gone {
				return
			}
			p.poured++
			r.addLine(Line{kind: Narrate, text: "the bartender sets " + p.drink + " in front of " + p.name + suffix})
		})
	})
}

func containsAny(s string, subs ...string) bool {
	for _, x := range subs {
		if strings.Contains(s, x) {
			return true
		}
	}
	return false
}

func (r *Room) keeperReply(p *Patron, text string) {
	t := strings.ToLower(text)

	if containsAny(t, "another", "refill", "one more", "same again") {
		time.AfterFunc(1200*time.Millisecond, func() {
			r.mu.Lock()
			defer r.mu.Unlock()
			if p.gone {
				return
			}
			p.poured++
			r.addLine(Line{kind: Narrate, text: "the bartender sets " + p.drink + " in front of " + p.name + "."})
		})
		return
	}

	var reply string
	switch {
	case containsAny(t, "thank"):
		reply = "mm."
	case containsAny(t, "bye", "good night", "goodnight", "see you"):
		reply = "mind the step."
	case containsAny(t, "weather", "rain", "fog", "cold out"):
		reply = "it'll pass. it always passes. that's the trouble with it."
	case containsAny(t, "loopback"):
		reply = "the bar was here before the name was."
	case containsAny(t, "quiet", "slow", "empty"):
		reply = "it's the hour."
	case containsAny(t, "who are you", "your name"):
		reply = "the bartender."
	case containsAny(t, "home"):
		reply = "you're alright here a while yet."
	case containsAny(t, "work", "job"):
		reply = "leave it at the door if you can. most can't."
	case containsAny(t, "why"):
		reply = "couldn't say."
	case strings.HasSuffix(t, "?") && rand.Float64() < 0.4:
		reply = "couldn't say."
	default:
		return
	}

	r.mu.Lock()
	if time.Now().Before(r.keeperBusyTill) {
		r.mu.Unlock()
		return
	}
	r.keeperBusyTill = time.Now().Add(15 * time.Second)
	r.mu.Unlock()

	time.AfterFunc(900*time.Millisecond, func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		if p.gone {
			return
		}
		r.addLine(Line{kind: Keeper, text: reply})
	})
}
