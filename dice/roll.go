package dice

import (
	"fmt"
	"math/rand/v2"
	"sort"
)

// maxDice caps the number of dice in a single roll, including dice added by
// explosions, so that input like "999999999d6" or "100d2!" stays bounded.
const maxDice = 1000

// ModifierKind identifies a modifier applied to a set of dice.
type ModifierKind int

const (
	ModExplode  ModifierKind = iota // roll an extra die for each die showing its maximum
	ModKeepHigh                     // keep the N highest dice, drop the rest
	ModKeepLow                      // keep the N lowest dice, drop the rest
	ModDropHigh                     // drop the N highest dice
	ModDropLow                      // drop the N lowest dice
)

// Modifier is a single modifier on a dice roll. N is unused for ModExplode.
type Modifier struct {
	Kind ModifierKind
	N    int
}

// Die is the outcome of one physical die.
type Die struct {
	Value    int
	Dropped  bool // excluded from the total by a keep/drop modifier
	Exploded bool // showed its maximum and caused an extra die to be rolled
}

// DiceRoll is the outcome of one dice term such as "4d6kl2".
type DiceRoll struct {
	Notation string // source text of the term, filled in by the parser
	Count    int    // number of dice requested, before explosions
	Sides    int
	Dice     []Die // every die rolled, in order, including dropped ones
	Total    int   // sum of the dice that were not dropped
}

// Roller rolls dice using a source of randomness.
type Roller struct {
	// intn returns a uniform random int in [0, n).
	intn func(n int) int
}

// NewRoller returns a Roller backed by the default random source.
func NewRoller() *Roller {
	return &Roller{intn: rand.IntN}
}

// Roll rolls count dice with the given number of sides and applies mods.
// Explosions happen as the dice are rolled; keep/drop modifiers are then
// applied in order, each one considering only the dice still kept.
func (r *Roller) Roll(count, sides int, mods []Modifier) (DiceRoll, error) {
	if count < 1 {
		return DiceRoll{}, fmt.Errorf("must roll at least 1 die, got %d", count)
	}
	if count > maxDice {
		return DiceRoll{}, fmt.Errorf("cannot roll more than %d dice, got %d", maxDice, count)
	}
	if sides < 1 {
		return DiceRoll{}, fmt.Errorf("dice must have at least 1 side, got %d", sides)
	}

	explode := false
	for _, m := range mods {
		if m.Kind == ModExplode {
			explode = true
		}
	}
	if explode && sides < 2 {
		return DiceRoll{}, fmt.Errorf("cannot explode a d%d: it would explode forever", sides)
	}

	roll := DiceRoll{Count: count, Sides: sides}

	// pending counts dice still to be rolled; each explosion adds one more.
	for pending := count; pending > 0 && len(roll.Dice) < maxDice; pending-- {
		die := Die{Value: r.intn(sides) + 1}
		if explode && die.Value == sides && len(roll.Dice)+pending < maxDice {
			die.Exploded = true
			pending++
		}
		roll.Dice = append(roll.Dice, die)
	}

	for _, m := range mods {
		kept := roll.keptCount()
		switch m.Kind {
		case ModKeepHigh:
			roll.drop(kept-m.N, false)
		case ModKeepLow:
			roll.drop(kept-m.N, true)
		case ModDropHigh:
			roll.drop(m.N, true)
		case ModDropLow:
			roll.drop(m.N, false)
		}
	}

	for _, d := range roll.Dice {
		if !d.Dropped {
			roll.Total += d.Value
		}
	}
	return roll, nil
}

func (r *DiceRoll) keptCount() int {
	n := 0
	for _, d := range r.Dice {
		if !d.Dropped {
			n++
		}
	}
	return n
}

// drop marks up to n of the kept dice as dropped, taking the highest values
// if highest is true and the lowest otherwise. Ties go to the earliest die.
func (r *DiceRoll) drop(n int, highest bool) {
	if n <= 0 {
		return
	}
	var kept []int // indices into r.Dice
	for i, d := range r.Dice {
		if !d.Dropped {
			kept = append(kept, i)
		}
	}
	sort.SliceStable(kept, func(a, b int) bool {
		if highest {
			return r.Dice[kept[a]].Value > r.Dice[kept[b]].Value
		}
		return r.Dice[kept[a]].Value < r.Dice[kept[b]].Value
	})
	for _, i := range kept[:min(n, len(kept))] {
		r.Dice[i].Dropped = true
	}
}
