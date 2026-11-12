package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"roll/dice"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: %s <notation>\n\nexample: %s 4d6kh3 + 2\n", os.Args[0], os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	// Joining the arguments lets the notation be given unquoted: roll 2d6 + 3
	notation := strings.Join(flag.Args(), " ")
	if notation == "" {
		flag.Usage()
		os.Exit(2)
	}

	result, err := dice.Parse(notation)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", os.Args[0], err)
		os.Exit(1)
	}

	for _, roll := range result.Rolls {
		fmt.Printf("%s: %s = %d\n", roll.Notation, formatDice(roll.Dice), roll.Total)
	}
	fmt.Printf("Total: %d\n", result.Total)
}

// formatDice renders dice as "[6! (2) 4]": "!" marks a die that exploded and
// parentheses mark one that was dropped.
func formatDice(dice []dice.Die) string {
	parts := make([]string, len(dice))
	for i, d := range dice {
		s := strconv.Itoa(d.Value)
		if d.Exploded {
			s += "!"
		}
		if d.Dropped {
			s = "(" + s + ")"
		}
		parts[i] = s
	}
	return "[" + strings.Join(parts, " ") + "]"
}
