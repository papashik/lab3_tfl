package main

import (
	"bufio"
	"cmp"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"slices"
	"strings"
	"sync"
)

// _P means PERCENT OF
const (
	MAX_TEST_LEN              = 100
	NEXT_TERMINAL_IS_RANDOM_P = 10
	START_NOT_WITH_START_P    = 10
	WORD_FINISH_IN_FINAL_P    = 70
	WORD_FINISH_P             = 1
	EMPTY_TESTS_ENABLED       = false

	JSON_FORMAT    = "JSON"
	DEFAULT_FORMAT = "DEFAULT"

	STDOUT_FILE_NAME = "STDOUT"
	STDIN_FILE_NAME  = "STDIN"
)

var (
	TEST_COUNT          = 20
	INPUT_FILE_NAME     = STDIN_FILE_NAME
	OUTPUT_FILE_NAME    = STDOUT_FILE_NAME
	OUTPUT_FORMAT       = DEFAULT_FORMAT
	NECESSARY_POSITIVE  = false
	POSITIVE_PERCENTAGE = 50
	VERBOSE_OUTPUT      = false
	ALL_SYMBOLS         = false
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

func IsTerminal(s string) bool {
	return len(s) == 1 && s[0] >= 'a' && s[0] <= 'z'
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
	FIRST        MapNtTBool
	LAST         MapNtTBool
	FOLLOW       map[NonTerminal]map[Symbol]bool
	PRECEDE      map[Symbol]map[Terminal]bool
	bigramMap    map[Terminal]map[Terminal]bool
}

func NewGrammarFromInput() *Grammar {
	var file *os.File
	var err error
	if INPUT_FILE_NAME == STDIN_FILE_NAME {
		file = os.Stdin
	} else {
		file, err = os.Open(INPUT_FILE_NAME)
		if err != nil {
			panic(err)
		}
		defer file.Close()
	}
	var g = &Grammar{terminals: make(map[Terminal]bool), nonterminals: make(map[NonTerminal]bool)}
	var sc = bufio.NewScanner(file)
	var args []string
	var nt NonTerminal
Scanning:
	for sc.Scan() {
		if err = sc.Err(); err != nil {
			panic(err)
		}
		args = strings.Fields(sc.Text())
		if len(args) == 0 {
			continue
		}
		switch args[0] {
		case "END":
			break Scanning
		case "TLIST":
			for _, str := range args[2:] {
				g.terminals[Terminal(str)] = true
			}
		case "NTLIST":
			break
		default:
			// A -> B c
			nt = NonTerminal(args[0])
			g.nonterminals[nt] = true
			var symbols []Symbol
			for _, str := range args[2:] {
				if IsTerminal(str) {
					symbols = append(symbols, Terminal(str))
					g.terminals[Terminal(str)] = true
				} else {
					symbols = append(symbols, NonTerminal(str))
					g.nonterminals[NonTerminal(str)] = true
				}
			}
			g.rules = append(g.rules, Rule{nt, symbols})
		}
	}
	if ALL_SYMBOLS {
		for i := 'a'; i <= 'z'; i++ {
			g.terminals[Terminal(i)] = true
		}
	}
	return g
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

func (g *Grammar) IsTerminal(t Terminal) bool {
	return g.terminals[t]
}

func (g *Grammar) IsNonTerminal(n NonTerminal) bool {
	return g.nonterminals[n]
}

func (g *Grammar) RemoveLongRules() {
	var newRules = make([]Rule, 0, len(g.rules))
	var nontermNum = 1

	for _, rule := range g.rules {
		if len(rule.right) <= 2 {
			newRules = append(newRules, rule)
			continue
		}

		var nonterm = rule.left
		for i := 0; i < len(rule.right)-2; i++ {
			first := rule.right[i]
			newNonTerminal := NonTerminal(fmt.Sprintf("LONG_%d", nontermNum))
			g.nonterminals[newNonTerminal] = true
			nontermNum++

			newRules = append(newRules, Rule{
				left:  nonterm,
				right: []Symbol{first, newNonTerminal},
			})

			nonterm = newNonTerminal
		}

		newRules = append(newRules, Rule{
			left:  nonterm,
			right: []Symbol{rule.right[len(rule.right)-2], rule.right[len(rule.right)-1]},
		})
	}

	g.rules = newRules

	if VERBOSE_OUTPUT {
		fmt.Println("Grammar after removing long rules:")
		fmt.Println(g)
	}
}

func (g *Grammar) RemoveChainRules() {
Changed:
	var changed = false
	ruleMap := make(map[NonTerminal][]Rule)

	for _, rule := range g.rules {
		ruleMap[rule.left] = append(ruleMap[rule.left], rule)
	}

	var newRules []Rule

	for _, rule := range g.rules {
		if nonTerminal, ok := rule.right[0].(NonTerminal); ok && len(rule.right) == 1 {
			if rules, exists := ruleMap[nonTerminal]; exists {
				for _, r := range rules {
					newRules = append(newRules, Rule{left: rule.left, right: r.right})
					changed = true
				}
			}
		} else {
			newRules = append(newRules, rule)
		}
	}
	g.rules = newRules

	if changed {
		goto Changed
	}

	if VERBOSE_OUTPUT {
		fmt.Println("Grammar after removing chain rules:")
		fmt.Println(g)
	}
}

func (g *Grammar) RemoveUselessSymbols() {
	generating := make(map[NonTerminal]bool)
Changed:
	var changed = false
	for _, rule := range g.rules {
		if generating[rule.left] {
			continue
		}
		if AllTerminals(rule.right) || AllGenerating(rule.right, generating) {
			generating[rule.left] = true
			changed = true
		}
	}
	if changed {
		goto Changed
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

	for nt := range g.nonterminals {
		if !reachable[nt] || !generating[nt] {
			delete(g.nonterminals, nt)
		}
	}

	g.rules = newRules

	if VERBOSE_OUTPUT {
		fmt.Println("Grammar after removing useless symbols:")
		fmt.Println(g)
	}
}

func (g *Grammar) ToChomskyNormalForm() {
	g.RemoveLongRules()
	g.RemoveChainRules()
	g.RemoveUselessSymbols()

	var nontermNum = 1

	var newRules = make([]Rule, 0, len(g.rules))
	for _, rule := range g.rules {
		// "A -> a"
		if _, ok := rule.right[0].(Terminal); ok && len(rule.right) == 1 {
			newRules = append(newRules, rule)
			continue
		}

		// "A -> U1 U2", Ui can be terminal or nonterminal
		var newRight = make([]Symbol, 2)
		newRules = append(newRules, Rule{
			left:  rule.left,
			right: newRight,
		})
		for i := 0; i < len(rule.right); i++ {
			switch symbol := rule.right[i].(type) {
			case Terminal:
				// Создаём новый нетерминал
				newNonTerminal := NonTerminal(fmt.Sprintf("CNF_%d", nontermNum))
				g.nonterminals[newNonTerminal] = true
				nontermNum++
				newRules = append(newRules, Rule{
					left:  newNonTerminal,
					right: []Symbol{symbol},
				})
				newRight[i] = newNonTerminal
			case NonTerminal:
				newRight[i] = symbol
			}
		}

	}
	g.rules = newRules

	if VERBOSE_OUTPUT {
		fmt.Println("Grammar in CNF:")
		fmt.Println(g)
	}
}

func (g *Grammar) ComputeSets() {
	if g.FIRST == nil {
		g.ComputeFIRST()
	}
	if g.LAST == nil {
		g.ComputeLAST()
	}
	if g.FOLLOW == nil {
		g.ComputeFOLLOW()
	}
	if g.PRECEDE == nil {
		g.ComputePRECEDE()
	}
}

func (g *Grammar) ComputeFIRST() {
	FIRST := make(MapNtTBool)

	for nt := range g.nonterminals {
		FIRST[nt] = make(map[Terminal]bool)
	}

Changed:
	var changed = false
	for _, rule := range g.rules {
		A := rule.left
		switch sym := rule.right[0].(type) {
		case Terminal:
			if !FIRST[A][sym] {
				FIRST[A][sym] = true
				changed = true
			}
		case NonTerminal:
			for t := range FIRST[sym] {
				if !FIRST[A][t] {
					FIRST[A][t] = true
					changed = true
				}
			}
		}
	}
	if changed {
		goto Changed
	}

	g.FIRST = FIRST
}

func (g *Grammar) ComputeLAST() {
	LAST := make(MapNtTBool)

	for nt := range g.nonterminals {
		LAST[nt] = make(map[Terminal]bool)
	}

Changed:
	var changed = false
	for _, rule := range g.rules {
		A := rule.left
		curr := rule.right[len(rule.right)-1]

		switch currSym := curr.(type) {
		case Terminal:
			if !LAST[A][currSym] {
				LAST[A][currSym] = true
				changed = true
			}
		case NonTerminal:
			for t := range LAST[currSym] {
				if !LAST[A][t] {
					LAST[A][t] = true
					changed = true
				}
			}
		}
	}
	if changed {
		goto Changed
	}

	g.LAST = LAST
}

func (g *Grammar) ComputeFOLLOW() {
	/*
		We use type Symbol in FOLLOW set, because 4th condition contains rule "nt2 belongs to FOLLOW(nt1)",
		so we need to process also nonterminals
	*/
	FOLLOW := make(map[NonTerminal]map[Symbol]bool)

	for nt := range g.nonterminals {
		FOLLOW[nt] = make(map[Symbol]bool)
	}

	startSymbol := g.rules[0].left
	FOLLOW[startSymbol][Terminal("$")] = true

Changed:
	var changed = false
	for _, rule := range g.rules {
		A := rule.left
		for i, symbol := range rule.right {
			if B, ok := symbol.(NonTerminal); ok {
				// Если это последний символ, то в его множество FOLLOW добавляем элементы FOLLOW(A)
				if i == len(rule.right)-1 {
					for t := range FOLLOW[A] {
						if !FOLLOW[B][t] {
							FOLLOW[B][t] = true
							changed = true
						}
					}
					continue
				}
				nextSymbol := rule.right[i+1]
				switch next := nextSymbol.(type) {
				case Terminal:
					if !FOLLOW[B][next] {
						FOLLOW[B][next] = true
						changed = true
					}
				case NonTerminal:
					if !FOLLOW[B][next] {
						FOLLOW[B][next] = true
						changed = true
					}
					for t := range g.FIRST[next] {
						if !FOLLOW[B][t] {
							FOLLOW[B][t] = true
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

	g.FOLLOW = FOLLOW
}

func (g *Grammar) ComputePRECEDE() {
	PRECEDE := make(map[Symbol]map[Terminal]bool)

	for t := range g.terminals {
		PRECEDE[t] = make(map[Terminal]bool)
	}
	for nt := range g.nonterminals {
		PRECEDE[nt] = make(map[Terminal]bool)
	}

Changed:
	var changed = false

	for _, rule := range g.rules {
		for i := 0; i < len(rule.right); i++ {
			curr := rule.right[i]
			if i == 0 {
				for t := range PRECEDE[rule.left] {
					if !PRECEDE[curr][t] {
						PRECEDE[curr][t] = true
						changed = true
					}
				}
				continue
			}

			prev := rule.right[i-1]

			switch prevSym := prev.(type) {
			case Terminal:
				if !PRECEDE[curr][prevSym] {
					PRECEDE[curr][prevSym] = true
					changed = true
				}

			case NonTerminal:
				// Если перед текущим символом - нетерминал, добавляем его LAST в PRECEDE(A)
				for t := range g.LAST[prevSym] {
					if !PRECEDE[curr][t] {
						PRECEDE[curr][t] = true
						changed = true
					}

				}
			}

		}
	}
	if changed {
		goto Changed
	}

	g.PRECEDE = PRECEDE
}

func (g *Grammar) ComputeBigramMap() {
	g.ComputeSets()

	var result = make(map[Terminal]map[Terminal]bool)
	for t := range g.terminals {
		result[t] = make(map[Terminal]bool)
	}

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
				// 2. If some terminal t1 belongs to LAST(nt1) and t2 belongs to FOLLOW(nt1)
				if g.LAST[nt1][t1] && g.FOLLOW[nt1][t2] {
					result[t1][t2] = true
				}
				// 3. If some terminal t1 belongs to PRECEDE(nt1) and t2 belongs to FIRST(nt1)
				if g.PRECEDE[nt1][t1] && g.FIRST[nt1][t2] {
					result[t1][t2] = true
				}
			}
		}

		// 4. If nt2 belongs to FOLLOW(nt1) and some terminal t1 belongs to LAST(nt1) and t2 belongs to FIRST(nt2)
		for nt2 := range g.nonterminals {
			if g.FOLLOW[nt1][nt2] {
				for t1 := range g.terminals {
					for t2 := range g.terminals {
						if g.LAST[nt1][t1] && g.FIRST[nt2][t2] {
							result[t1][t2] = true
						}
					}
				}
			}
		}
	}

	g.bigramMap = result
	if VERBOSE_OUTPUT {
		fmt.Println("Bigram map:")
		for k, v := range result {
			fmt.Printf("%v : %v\n", k, v)
		}
	}
}

func (g *Grammar) CYKParse(input string) bool {
	// Grammar must be in CNF

	var n = len(input)
	if n == 0 {
		return false
	}

	table := make([][]map[NonTerminal]bool, n)
	for i := range table {
		table[i] = make([]map[NonTerminal]bool, n)
		for j := range table[i] {
			table[i][j] = make(map[NonTerminal]bool)
		}
	}

	for i, char := range input {
		t := Terminal(char) // Символ в строке как Terminal
		for _, rule := range g.rules {
			if len(rule.right) == 1 {
				if term, ok := rule.right[0].(Terminal); ok && term == t {
					table[i][i][rule.left] = true
				}
			}
		}
	}

	for span := 2; span <= n; span++ {
		for i := 0; i <= n-span; i++ {
			j := i + span - 1
			for k := i; k < j; k++ {
				for _, rule := range g.rules {
					if len(rule.right) == 2 {
						B, _ := rule.right[0].(NonTerminal)
						C, _ := rule.right[1].(NonTerminal)
						if table[i][k][B] && table[k+1][j][C] {
							table[i][j][rule.left] = true
						}
					}
				}
			}
		}
	}

	// Если в стартовом узле содержится начальный символ грамматики, то входная строка принадлежит языку
	startSymbol := g.rules[0].left
	return table[0][n-1][startSymbol]
}

func (g *Grammar) GenerateTests() []Test {
	var tests = make([]Test, 0, TEST_COUNT)
	var positiveCount = TEST_COUNT * POSITIVE_PERCENTAGE / 100
	g.ToChomskyNormalForm()
	g.ComputeBigramMap()

	var wgReader, wgTest sync.WaitGroup
	wgReader.Add(1)
	wgTest.Add(TEST_COUNT)
	var testChannel = make(chan Test, TEST_COUNT)
	var testsMap sync.Map
	var counter = 1
	go func() {
		for test := range testChannel {
			if _, ok := testsMap.Load(test.Question); ok {
				go g.NewTest(testChannel, test.Answer, &testsMap)
			} else {
				tests = append(tests, test)
				testsMap.Store(test.Question, true)
				if VERBOSE_OUTPUT {
					fmt.Printf("Test %2v: %5v <- %v\n", counter, test.Answer, test.Question)
				}
				counter++
				wgTest.Done()
			}
		}
		wgReader.Done()
	}()

	var positive = false
	for i := 0; i < TEST_COUNT; i++ {
		if NECESSARY_POSITIVE && i == TEST_COUNT-positiveCount {
			positive = true
		}
		go g.NewTest(testChannel, positive, &testsMap)
	}
	wgTest.Wait()
	close(testChannel)
	wgReader.Wait()

	slices.SortFunc(tests, func(a, b Test) int {
		return cmp.Compare(a.Question, b.Question)
	})
	return tests
}

func (g *Grammar) NewTest(testChannel chan Test, positive bool, testsMap *sync.Map) {
	var question strings.Builder
	var isFinal = g.LAST[g.rules[0].left]
	var t Terminal
	var possibleTerminals = g.FIRST[g.rules[0].left]
	if !positive && Random(START_NOT_WITH_START_P) {
		possibleTerminals = g.terminals
	}
	for i := 0; i < MAX_TEST_LEN; i++ {
		if !positive && Random(NEXT_TERMINAL_IS_RANDOM_P) {
			t = PickRandomKey(g.terminals)
		} else {
			if len(possibleTerminals) == 0 {
				break
			}
			_, ok := testsMap.Load(question.String())
			if !ok && (!positive && Random(WORD_FINISH_P) || isFinal[t] && Random(WORD_FINISH_IN_FINAL_P)) {
				break
			}
			t = PickRandomKey(possibleTerminals)
		}
		possibleTerminals = g.bigramMap[t]
		question.WriteString(t.String())
	}

	var qString = question.String()
	var answer = g.CYKParse(qString)
	if (!EMPTY_TESTS_ENABLED && len(qString) == 0) || NECESSARY_POSITIVE && answer != positive {
		go g.NewTest(testChannel, positive, testsMap)
	} else {
		testChannel <- Test{qString, answer}
	}
}

func WriteTestsToFile(tests []Test) {
	var file *os.File
	var err error
	if OUTPUT_FILE_NAME == STDOUT_FILE_NAME {
		file = os.Stdout
	} else {
		file, err = os.Create(OUTPUT_FILE_NAME)
		if err != nil {
			panic(err)
		}
	}

	switch OUTPUT_FORMAT {
	case JSON_FORMAT:
		var enc = json.NewEncoder(file)
		enc.SetIndent("", "\t")
		err = enc.Encode(tests)
		if err != nil {
			panic(err)
		}
	default:
		var w = bufio.NewWriter(file)
		var str string
		for _, t := range tests {
			if t.Answer {
				str = t.Question + " 1\n"
			} else {
				str = t.Question + " 0\n"
			}
			_, err = w.WriteString(str)
			if err != nil {
				panic(err)
			}
		}
		err = w.Flush()
		if err != nil {
			panic(err)
		}
	}

	err = file.Close()
	if err != nil {
		panic(err)
	}

	if OUTPUT_FILE_NAME != STDOUT_FILE_NAME {
		fmt.Println("Tests are written to file", OUTPUT_FILE_NAME)
	}
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

func PickRandomKey[K comparable, V any](m map[K]V) K {
	var ind = rand.Intn(len(m))
	for key := range m {
		if ind == 0 {
			return key
		}
		ind--
	}
	panic("Pick random key: unreachable")
}

func Random(percent int) bool {
	return rand.Intn(100) < percent
}

func main() {
	flag.IntVar(&TEST_COUNT, "count", TEST_COUNT, "Number of tests to generate")
	flag.StringVar(&INPUT_FILE_NAME, "input", INPUT_FILE_NAME, "Input file name or 'STDIN'")
	flag.StringVar(&OUTPUT_FILE_NAME, "output", OUTPUT_FILE_NAME, `Output file name or "STDOUT"`)
	flag.StringVar(&OUTPUT_FORMAT, "format", OUTPUT_FORMAT, `Tests output format ("JSON" or "DEFAULT")`)
	flag.BoolVar(&NECESSARY_POSITIVE, "necessary", NECESSARY_POSITIVE,
		`If set, percentage of positive tests will be satisfied at any performance cost.
Program can freeze and work slowly while looking for positive tests`)
	flag.IntVar(&POSITIVE_PERCENTAGE, "percent", POSITIVE_PERCENTAGE, "Percentage of positive tests")
	flag.BoolVar(&VERBOSE_OUTPUT, "verbose", VERBOSE_OUTPUT, "Verbose output in STDOUT")
	flag.BoolVar(&ALL_SYMBOLS, "allsymbols", ALL_SYMBOLS, "If set, all a-z symbols will be used to form tests")
	flag.Parse()

	var g = NewGrammarFromInput()
	tests := g.GenerateTests()
	WriteTestsToFile(tests)
}
