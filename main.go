package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Rule struct {
	left  NonTerminal
	right []Symbol
}

type Symbol interface {
	String() string
}

type Terminal string

func (t Terminal) String() string {
	return string(t)
}

type NonTerminal string

func (n NonTerminal) String() string {
	return string(n)
}

type Grammar struct {
	rules        []Rule
	terminals    []Terminal
	nonterminals []NonTerminal
}

func NewGrammarFromInput() Grammar {
	var g Grammar
	var sc = bufio.NewScanner(os.Stdin)
	var args []string
	for sc.Scan() {
		args = strings.Fields(sc.Text())
		if len(args) == 0 {
			continue
		}
		switch args[0] {
		case "rule":
			var symbols []Symbol
			for _, str := range args[4:] {
				symbols = append(symbols, NonTerminal(str))
			}
			g.rules = append(g.rules, Rule{NonTerminal(args[2]), symbols})
		case "T":
			g.terminals = append(g.terminals, Terminal(args[2]))
		case "NT":
			g.nonterminals = append(g.nonterminals, NonTerminal(args[2]))
		}
	}

	// checking whether symbols are terminals or not
	for _, rule := range g.rules {
		for i, symbol := range rule.right {
			t := Terminal(symbol.String())
			if g.IsTerminal(t) {
				rule.right[i] = t
			}
		}
	}

	return g
}

func (g *Grammar) IsTerminal(t Terminal) bool {
	for _, term := range g.terminals {
		if t == term {
			return true
		}
	}
	return false
}

func (g *Grammar) RemoveChainRules() {
	var changed bool
	ruleMap := make(map[NonTerminal][]Rule)

	for _, rule := range g.rules {
		ruleMap[rule.left] = append(ruleMap[rule.left], rule)
	}

	var newRules []Rule

	for _, rule := range g.rules {
		if len(rule.right) == 1 {
			if _, ok := rule.right[0].(NonTerminal); ok {
				nonTerminal := rule.right[0].(NonTerminal)
				if rules, exists := ruleMap[nonTerminal]; exists {
					for _, r := range rules {
						newRules = append(newRules, Rule{left: rule.left, right: r.right})
						changed = true
					}
				}
				continue
			}
		}
		newRules = append(newRules, rule)
	}
	g.rules = newRules

	if changed {
		g.RemoveChainRules()
	}
}

func (g *Grammar) RemoveUselessSymbols() {
	generating := make(map[NonTerminal]bool)
	for {
		newGenerating := make(map[NonTerminal]bool)
		for _, rule := range g.rules {
			if generating[rule.left] {
				continue
			}
			if allTerminals(rule.right) || allGenerating(rule.right, generating) {
				newGenerating[rule.left] = true
			}
		}
		if len(newGenerating) == 0 {
			break
		}
		for k := range newGenerating {
			generating[k] = true
		}
	}

	reachable := make(map[NonTerminal]bool)
	reachable[g.nonterminals[0]] = true
	queue := []NonTerminal{g.nonterminals[0]}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, rule := range g.rules {
			if rule.left == current {
				for _, sym := range rule.right {
					if nt, ok := sym.(NonTerminal); ok && !reachable[nt] {
						reachable[nt] = true
						queue = append(queue, nt)
					}
				}
			}
		}
	}

	var newRules []Rule
RulesLoop:
	for _, rule := range g.rules {
		if reachable[rule.left] && generating[rule.left] {
			for _, sym := range rule.right {
				// Если справа есть непорождающий
				if nt, ok := sym.(NonTerminal); ok && !generating[nt] {
					continue RulesLoop
				}
			}
			newRules = append(newRules, rule)
		}
	}

	g.rules = newRules
}

func allTerminals(symbols []Symbol) bool {
	for _, sym := range symbols {
		if _, ok := sym.(Terminal); !ok {
			return false
		}
	}
	return true
}

func allGenerating(symbols []Symbol, generating map[NonTerminal]bool) bool {
	for _, sym := range symbols {
		if nt, ok := sym.(NonTerminal); ok && !generating[nt] {
			return false
		}
	}
	return true
}

func main() {
	/*var g = Grammar{
		rules: []Rule{
			{left: "S", right: []Symbol{NonTerminal("A")}},
			{left: "A", right: []Symbol{Terminal('a')}},
			{left: "B", right: []Symbol{NonTerminal("C")}},
			{left: "C", right: []Symbol{Terminal('b')}},
			{left: "D", right: []Symbol{NonTerminal("E")}}, // Непорождающий символ
			{left: "E", right: []Symbol{NonTerminal("F")}}, // Недостижимый символ
			{left: "F", right: []Symbol{Terminal('c')}},
		},
		terminals:    []Terminal{"a", "b", "c", "d", "e", "f"},
		nonterminals: []NonTerminal{"S", "A", "B", "C", "D", "E", "F"},
	}*/
	var g = NewGrammarFromInput()

	fmt.Println("Grammar:")
	for _, rule := range g.rules {
		fmt.Printf("%s -> %v\n", rule.left.String(), rule.right)
	}

	g.RemoveChainRules()
	fmt.Println("After removing chain rules:")
	for _, rule := range g.rules {
		fmt.Printf("%s -> %v\n", rule.left.String(), rule.right)
	}

	g.RemoveUselessSymbols()
	fmt.Println("After removing useless symbols:")
	for _, rule := range g.rules {
		fmt.Printf("%s -> %v\n", rule.left.String(), rule.right)
	}

}
