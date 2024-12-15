package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strings"
)

const (
	MAXTESTLEN = 100
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

type MapNtTBool map[NonTerminal]map[Terminal]bool

type Test struct {
	Question string
	Answer   bool
}

type Grammar struct {
	rules        []Rule
	terminals    map[Terminal]bool
	nonterminals map[NonTerminal]bool
}

func (g *Grammar) String() string {
	var sb strings.Builder
	for _, rule := range g.rules {
		_, err := fmt.Fprintf(&sb, "%s -> %v\n", rule.left, rule.right)
		if err != nil {
			panic(err)
		}
	}
	return sb.String()
}

func NewGrammarFromInput() *Grammar {
	var g = &Grammar{terminals: make(map[Terminal]bool), nonterminals: make(map[NonTerminal]bool)}
	var sc = bufio.NewScanner(os.Stdin)
	var args []string
	var rulesToProcess [][]string
	for sc.Scan() {
		args = strings.Fields(sc.Text())
		if len(args) == 0 {
			continue
		}
		switch args[0] {
		case "rule":
			// rule ::= A -> B C
			rulesToProcess = append(rulesToProcess, args[2:])
		case "T":
			// T := a
			g.terminals[Terminal(args[2])] = true
		case "NT":
			// NT := A
			g.nonterminals[NonTerminal(args[2])] = true
		case "TLIST":
			// TLIST = a b c
			for _, str := range args[2:] {
				g.terminals[Terminal(str)] = true
			}
		case "NTLIST":
			// NTLIST = A B C D
			for _, str := range args[2:] {
				g.nonterminals[NonTerminal(str)] = true
			}
		default:
			// A -> B C
			rulesToProcess = append(rulesToProcess, args)
		}
	}

	// processing rules with knowledge about what symbol is terminal
	for _, rule := range rulesToProcess {
		var symbols []Symbol
		for _, str := range rule[2:] {
			if g.IsTerminal(Terminal(str)) {
				symbols = append(symbols, Terminal(str))
			} else {
				symbols = append(symbols, NonTerminal(str))
			}
		}
		g.rules = append(g.rules, Rule{NonTerminal(rule[0]), symbols})
	}

	return g
}

func (g *Grammar) IsTerminal(t Terminal) bool {
	return g.terminals[t]
}

func (g *Grammar) IsNonTerminal(n NonTerminal) bool {
	return g.nonterminals[n]
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
			if AllTerminals(rule.right) || AllGenerating(rule.right, generating) {
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
	reachable[g.rules[0].left] = true
	queue := []NonTerminal{g.rules[0].left}
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
				// Если справа есть непорождающий символ, выкидываем правило
				if nt, ok := sym.(NonTerminal); ok && !generating[nt] {
					continue RulesLoop
				}
			}
			newRules = append(newRules, rule)
		}
	}

	g.rules = newRules
}

func (g *Grammar) ComputeFirst() MapNtTBool {
	first := make(MapNtTBool)

	for nt := range g.nonterminals {
		first[nt] = make(map[Terminal]bool)
	}

Changed:
	var changed = false
	for _, rule := range g.rules {
		A := rule.left
		switch sym := rule.right[0].(type) {
		case Terminal:
			if !first[A][sym] {
				first[A][sym] = true
				changed = true
			}
		case NonTerminal:
			for t := range first[sym] {
				if !first[A][t] {
					first[A][t] = true
					changed = true
				}
			}
		}
	}
	if changed {
		goto Changed
	}

	return first
}

func (g *Grammar) ComputeLast() MapNtTBool {
	last := make(MapNtTBool)

	for nt := range g.nonterminals {
		last[nt] = make(map[Terminal]bool)
	}

Changed:
	var changed = false
	for _, rule := range g.rules {
		A := rule.left
		curr := rule.right[len(rule.right)-1]

		switch currSym := curr.(type) {
		case Terminal:
			if !last[A][currSym] {
				last[A][currSym] = true
				changed = true
			}
		case NonTerminal:
			for t := range last[currSym] {
				if !last[A][t] {
					last[A][t] = true
					changed = true
				}
			}
		}
	}
	if changed {
		goto Changed
	}

	return last
}

func (g *Grammar) ComputeFollow(firstSets MapNtTBool) map[NonTerminal]map[Symbol]bool {
	/*
		We use type Symbol in follow set, because 4th condition contains rule "nt2 belongs to follow(nt1)",
		so we need to process also nonterminals
	*/
	follow := make(map[NonTerminal]map[Symbol]bool)

	for nt := range g.nonterminals {
		follow[nt] = make(map[Symbol]bool)
	}

	startSymbol := g.rules[0].left
	follow[startSymbol][Terminal("$")] = true

Changed:
	var changed = false
	for _, rule := range g.rules {
		A := rule.left
		for i, symbol := range rule.right {
			if B, ok := symbol.(NonTerminal); ok {
				// Если это последний символ, то в его множество follow добавляем элементы follow(A)
				if i == len(rule.right)-1 {
					for t := range follow[A] {
						if !follow[B][t] {
							follow[B][t] = true
							changed = true
						}
					}
					continue
				}
				nextSymbol := rule.right[i+1]
				switch next := nextSymbol.(type) {
				case Terminal:
					if !follow[B][next] {
						follow[B][next] = true
						changed = true
					}
				case NonTerminal:
					if !follow[B][next] {
						follow[B][next] = true
						changed = true
					}
					for t := range firstSets[next] {
						if !follow[B][t] {
							follow[B][t] = true
							changed = true
						}
					}
				}
			}
		}
	}
	if changed {
		goto Changed
	}

	return follow
}

func (g *Grammar) ComputePrecede(lastSets MapNtTBool) map[Symbol]map[Terminal]bool {
	precede := make(map[Symbol]map[Terminal]bool)

	for t := range g.terminals {
		precede[t] = make(map[Terminal]bool)
	}
	for nt := range g.nonterminals {
		precede[nt] = make(map[Terminal]bool)
	}

Changed:
	var changed = false

	for _, rule := range g.rules {
		for i := 0; i < len(rule.right); i++ {
			curr := rule.right[i]
			if i == 0 {
				for t := range precede[rule.left] {
					if !precede[curr][t] {
						precede[curr][t] = true
						changed = true
					}
				}
				continue
			}

			prev := rule.right[i-1]

			switch prevSym := prev.(type) {
			case Terminal:
				if !precede[curr][prevSym] {
					precede[curr][prevSym] = true
					changed = true
				}

			case NonTerminal:
				// Если перед текущим символом - нетерминал, добавляем его LAST в PRECEDE(A)
				for t := range lastSets[prevSym] {
					if !precede[curr][t] {
						precede[curr][t] = true
						changed = true
					}

				}
			}

		}
	}
	if changed {
		goto Changed
	}

	return precede
}

func (g *Grammar) ComputeBigramMap() map[Terminal]map[Terminal]bool {
	var result = make(map[Terminal]map[Terminal]bool)
	for t := range g.terminals {
		result[t] = make(map[Terminal]bool)
	}

	var first = g.ComputeFirst()
	var last = g.ComputeLast()
	var follow = g.ComputeFollow(first)
	var precede = g.ComputePrecede(last)

	// 1. If there is a sequence "t1 t2"
	for _, rule := range g.rules {
		for i := 1; i < len(rule.right); i++ {
			t1, ok1 := rule.right[i].(Terminal)
			t2, ok2 := rule.right[i-1].(Terminal)
			if ok1 && ok2 {
				result[t1][t2] = true
			}
		}
	}

	for nt1 := range g.nonterminals {
		for t1 := range g.terminals {
			for t2 := range g.terminals {
				// 2. If some terminal t1 belongs to last(nt1) and t2 belongs to follow(nt1)
				if last[nt1][t1] && follow[nt1][t2] {
					result[t1][t2] = true
				}
				// 3. If some terminal t1 belongs to precede(nt1) and t2 belongs to first(nt1)
				if precede[nt1][t1] && first[nt1][t2] {
					result[t1][t2] = true
				}
			}
		}

		// 4. If nt2 belongs to follow(nt1) and some terminal t1 belongs to last(nt1) and t2 belongs to first(nt2)
		for nt2 := range g.nonterminals {
			if follow[nt1][nt2] {
				for t1 := range g.terminals {
					for t2 := range g.terminals {
						if last[nt1][t1] && first[nt2][t2] {
							result[t1][t2] = true
						}
					}
				}
			}
		}
	}

	return result
}

func (g *Grammar) ComputeAnswer(question string) bool {

	return rand.Intn(100) < 50
}

func (g *Grammar) GenerateTests(n int) []Test {
	var tests []Test
	var bigramMap = g.ComputeBigramMap()
	for i := 0; i < n; i++ {
		tests = append(tests, g.NewTest(bigramMap))
	}
	return tests
}

func (g *Grammar) NewTest(bigramMap map[Terminal]map[Terminal]bool) Test {
	var length = rand.Intn(1 + MAXTESTLEN)
	var question strings.Builder
	question.Grow(length)
	var t Terminal
	var possibleTerminals = g.terminals
	for i := 0; i < length; i++ {
		if len(possibleTerminals) == 0 {
			break
		}
		t = pickRandomKey(possibleTerminals)
		possibleTerminals = bigramMap[t]
		question.WriteString(t.String())
	}

	var qString = question.String()
	return Test{qString, g.ComputeAnswer(qString)}
}

func AllTerminals(symbols []Symbol) bool {
	for _, sym := range symbols {
		if _, ok := sym.(Terminal); !ok {
			return false
		}
	}
	return true
}

func AllGenerating(symbols []Symbol, generating map[NonTerminal]bool) bool {
	for _, sym := range symbols {
		if nt, ok := sym.(NonTerminal); ok && !generating[nt] {
			return false
		}
	}
	return true
}

func pickRandomKey[K comparable, V any](m map[K]V) (ret K) {
	var ind = rand.Intn(len(m))
	for key := range m {
		if ind == 0 {
			return key
		}
		ind--
	}
	return
}

func main() {
	var g = NewGrammarFromInput()

	fmt.Println("Grammar:")
	fmt.Println(g)

	g.RemoveChainRules()
	fmt.Println("After removing chain rules:")
	fmt.Println(g)

	g.RemoveUselessSymbols()
	fmt.Println("After removing useless symbols:")
	fmt.Println(g)

	fmt.Println("Tests:")
	tests := g.GenerateTests(10)
	for _, t := range tests {
		fmt.Printf("%5v <- %v\n", t.Answer, t.Question)
	}
}
