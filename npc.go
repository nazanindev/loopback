package main

import (
	"math/rand"
	"time"
)

// The regulars. They keep their own hours; the roster for any given hour of a
// given night is canonical, so if marlow was in at eleven, he was in at eleven
// for everyone.
type NPC struct {
	name    string
	enters  string
	leaves  string
	mutters []string
	pEve    float64 // 6pm–1am
	pNight  float64 // 1am–6am
	pDay    float64 // the rest, for people with nowhere else to be
	present bool
}

func theRegulars() []*NPC {
	return []*NPC{
		{
			name:   "marlow",
			enters: "comes in and takes his usual stool without looking",
			leaves: "leaves the exact money on the counter and goes",
			mutters: []string{
				"some nights the rain sounds like static. some nights it's the other way.",
				"i used to fix things for a living. can't say what kind anymore.",
				"the bridge traffic's changed. don't ask me how i can tell.",
			},
			pEve: 0.75, pNight: 0.30, pDay: 0.20,
		},
		{
			name:   "june",
			enters: "comes in still wearing her badge, and turns it around",
			leaves: "checks her phone, winces, and goes",
			mutters: []string{
				"twelve hours. don't ask.",
				"the machines were kind tonight, for once.",
				"if it pages after midnight it can learn patience.",
			},
			pEve: 0.30, pNight: 0.70, pDay: 0.15,
		},
		{
			name:   "the tall one",
			enters: "ducks under the doorframe and nods to nobody",
			leaves: "unfolds toward the door and is gone",
			mutters: []string{
				"hm.",
				"…",
			},
			pEve: 0.40, pNight: 0.40, pDay: 0.30,
		},
		{
			name:   "ada",
			enters: "comes in carrying a laptop she has no intention of opening",
			leaves: "closes the laptop she never opened and heads out",
			mutters: []string{
				"you can hear a fan fail two days before it does. same with people.",
				"i named all the servers after birds. they still went down.",
				"nothing is ever down. it's just up somewhere you can't see.",
			},
			pEve: 0.50, pNight: 0.45, pDay: 0.25,
		},
		{
			name:   "moss",
			enters: "comes in beaming about nothing in particular",
			leaves: "waves at the whole room and goes",
			mutters: []string{
				"tomorrow's looking alright, i think.",
				"you ever just watch the steam come off a kettle? good stuff.",
			},
			pEve: 0.50, pNight: 0.20, pDay: 0.40,
		},
		{
			name:   "the regular",
			enters: "is suddenly on his stool, as if he had always been there",
			leaves: "is gone, though nobody saw the door",
			mutters: []string{
				"same again.",
				"it was busier once. or i was.",
			},
			pEve: 0.80, pNight: 0.60, pDay: 0.50,
		},
		{
			name:   "a stranger",
			enters: "pushes the door like they expected it locked",
			leaves: "leaves without touching their glass",
			mutters: []string{
				"wrong bar.",
				"staying anyway.",
			},
			pEve: 0.12, pNight: 0.12, pDay: 0.12,
		},
	}
}

// rosterCheck settles who is in this hour. Callers hold r.mu (or nothing is
// running yet). Announcements are suppressed on the first call so the opening
// crowd is simply discovered in place.
func (r *Room) rosterCheck(silent bool) {
	hour := time.Now().Hour()
	for i, n := range r.regulars {
		rng := rand.New(rand.NewSource(nightSeed()*1000 + int64(hour)*37 + int64(i)))
		p := n.pDay
		switch {
		case hour >= 18 || hour == 0:
			p = n.pEve
		case hour >= 1 && hour < 6:
			p = n.pNight
		}
		want := rng.Float64() < p
		if want != n.present {
			n.present = want
			if !silent {
				if want {
					r.addLine(Line{kind: Narrate, text: n.name + " " + n.enters + "."})
				} else {
					r.addLine(Line{kind: Narrate, text: n.name + " " + n.leaves + "."})
				}
			}
		}
	}
}

// ambient is the room breathing: the keeper fidgets, the regulars mutter.
// Callers hold r.mu.
func (r *Room) ambient() {
	if rand.Float64() < 0.10 && time.Now().After(r.keeperBusyTill) {
		r.addLine(Line{kind: Narrate, text: keeperIdle[rand.Intn(len(keeperIdle))]})
		r.keeperBusyTill = time.Now().Add(20 * time.Second)
	}
	for _, n := range r.regulars {
		if n.present && rand.Float64() < 0.04 {
			r.addLine(Line{kind: Speech, who: n.name, text: n.mutters[rand.Intn(len(n.mutters))]})
		}
	}
}
