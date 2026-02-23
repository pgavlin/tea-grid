package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

// --- Tokenizer ---

type tokenKind int

const (
	tokNumber tokenKind = iota
	tokString
	tokCellRef // e.g. A1
	tokRange   // e.g. A1:B5
	tokFunc    // e.g. SUM
	tokPlus
	tokMinus
	tokStar
	tokSlash
	tokPercent
	tokCaret
	tokLParen
	tokRParen
	tokComma
	tokColon
	tokEOF
)

type token struct {
	kind tokenKind
	text string
	num  float64
}

func tokenize(input string) ([]token, error) {
	var tokens []token
	i := 0
	runes := []rune(input)

	for i < len(runes) {
		ch := runes[i]

		// Skip whitespace
		if unicode.IsSpace(ch) {
			i++
			continue
		}

		// Number literal
		if unicode.IsDigit(ch) || (ch == '.' && i+1 < len(runes) && unicode.IsDigit(runes[i+1])) {
			start := i
			for i < len(runes) && (unicode.IsDigit(runes[i]) || runes[i] == '.') {
				i++
			}
			text := string(runes[start:i])
			num, err := strconv.ParseFloat(text, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid number: %s", text)
			}
			tokens = append(tokens, token{kind: tokNumber, text: text, num: num})
			continue
		}

		// String literal (in double quotes)
		if ch == '"' {
			i++
			start := i
			for i < len(runes) && runes[i] != '"' {
				i++
			}
			if i >= len(runes) {
				return nil, fmt.Errorf("unterminated string")
			}
			text := string(runes[start:i])
			i++ // skip closing quote
			tokens = append(tokens, token{kind: tokString, text: text})
			continue
		}

		// Cell reference, range, or function name
		if unicode.IsLetter(ch) {
			start := i
			for i < len(runes) && (unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i])) {
				i++
			}
			text := string(runes[start:i])

			// Check for range: e.g. A1:B5
			if i < len(runes) && runes[i] == ':' {
				// Peek ahead for another cell ref
				j := i + 1
				for j < len(runes) && (unicode.IsLetter(runes[j]) || unicode.IsDigit(runes[j])) {
					j++
				}
				endText := string(runes[i+1 : j])
				if isCellRef(text) && isCellRef(endText) {
					tokens = append(tokens, token{kind: tokRange, text: text + ":" + endText})
					i = j
					continue
				}
			}

			// Check if it's a function name (followed by open paren)
			if i < len(runes) && runes[i] == '(' && !isCellRef(text) {
				tokens = append(tokens, token{kind: tokFunc, text: strings.ToUpper(text)})
				continue
			}

			// Cell reference
			if isCellRef(text) {
				tokens = append(tokens, token{kind: tokCellRef, text: strings.ToUpper(text)})
				continue
			}

			// Unknown identifier - treat as function name anyway
			tokens = append(tokens, token{kind: tokFunc, text: strings.ToUpper(text)})
			continue
		}

		// Operators and punctuation
		switch ch {
		case '+':
			tokens = append(tokens, token{kind: tokPlus, text: "+"})
		case '-':
			tokens = append(tokens, token{kind: tokMinus, text: "-"})
		case '*':
			tokens = append(tokens, token{kind: tokStar, text: "*"})
		case '/':
			tokens = append(tokens, token{kind: tokSlash, text: "/"})
		case '%':
			tokens = append(tokens, token{kind: tokPercent, text: "%"})
		case '^':
			tokens = append(tokens, token{kind: tokCaret, text: "^"})
		case '(':
			tokens = append(tokens, token{kind: tokLParen, text: "("})
		case ')':
			tokens = append(tokens, token{kind: tokRParen, text: ")"})
		case ',':
			tokens = append(tokens, token{kind: tokComma, text: ","})
		case ':':
			tokens = append(tokens, token{kind: tokColon, text: ":"})
		default:
			return nil, fmt.Errorf("unexpected character: %c", ch)
		}
		i++
	}

	tokens = append(tokens, token{kind: tokEOF})
	return tokens, nil
}

// isCellRef checks if text is a cell reference like A1, B12, AA5.
func isCellRef(text string) bool {
	if len(text) < 2 {
		return false
	}
	i := 0
	runes := []rune(text)
	for i < len(runes) && unicode.IsLetter(runes[i]) {
		i++
	}
	if i == 0 || i == len(runes) {
		return false
	}
	for i < len(runes) {
		if !unicode.IsDigit(runes[i]) {
			return false
		}
		i++
	}
	return true
}

// --- AST ---

type exprNode interface {
	exprNode()
}

type (
	numberLit   struct{ val float64 }
	stringLit   struct{ val string }
	cellRefExpr struct{ ref string } // e.g. "A1"
	rangeExpr   struct{ from, to string }
	binaryExpr  struct {
		op    tokenKind
		left  exprNode
		right exprNode
	}
)
type unaryExpr struct {
	op      tokenKind
	operand exprNode
}
type funcCall struct {
	name string
	args []exprNode
}

func (numberLit) exprNode()   {}
func (stringLit) exprNode()   {}
func (cellRefExpr) exprNode() {}
func (rangeExpr) exprNode()   {}
func (binaryExpr) exprNode()  {}
func (unaryExpr) exprNode()   {}
func (funcCall) exprNode()    {}

// --- Parser ---

type parser struct {
	tokens []token
	pos    int
}

func parse(input string) (exprNode, error) {
	tokens, err := tokenize(input)
	if err != nil {
		return nil, err
	}
	p := &parser{tokens: tokens}
	node, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if p.peek().kind != tokEOF {
		return nil, fmt.Errorf("unexpected token: %s", p.peek().text)
	}
	return node, nil
}

func (p *parser) peek() token {
	if p.pos >= len(p.tokens) {
		return token{kind: tokEOF}
	}
	return p.tokens[p.pos]
}

func (p *parser) advance() token {
	t := p.peek()
	p.pos++
	return t
}

func (p *parser) expect(kind tokenKind) (token, error) {
	t := p.advance()
	if t.kind != kind {
		return t, fmt.Errorf("expected %d, got %s", kind, t.text)
	}
	return t, nil
}

// parseExpr handles addition and subtraction (lowest precedence).
func (p *parser) parseExpr() (exprNode, error) {
	left, err := p.parseTerm()
	if err != nil {
		return nil, err
	}
	for p.peek().kind == tokPlus || p.peek().kind == tokMinus {
		op := p.advance()
		right, err := p.parseTerm()
		if err != nil {
			return nil, err
		}
		left = binaryExpr{op: op.kind, left: left, right: right}
	}
	return left, nil
}

// parseTerm handles multiplication and division.
func (p *parser) parseTerm() (exprNode, error) {
	left, err := p.parsePower()
	if err != nil {
		return nil, err
	}
	for p.peek().kind == tokStar || p.peek().kind == tokSlash || p.peek().kind == tokPercent {
		op := p.advance()
		right, err := p.parsePower()
		if err != nil {
			return nil, err
		}
		left = binaryExpr{op: op.kind, left: left, right: right}
	}
	return left, nil
}

// parsePower handles exponentiation (right-associative).
func (p *parser) parsePower() (exprNode, error) {
	base, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	if p.peek().kind == tokCaret {
		p.advance()
		exp, err := p.parsePower() // right-associative
		if err != nil {
			return nil, err
		}
		return binaryExpr{op: tokCaret, left: base, right: exp}, nil
	}
	return base, nil
}

// parseUnary handles unary minus.
func (p *parser) parseUnary() (exprNode, error) {
	if p.peek().kind == tokMinus {
		p.advance()
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return unaryExpr{op: tokMinus, operand: operand}, nil
	}
	if p.peek().kind == tokPlus {
		p.advance()
		return p.parseUnary()
	}
	return p.parsePrimary()
}

// parsePrimary handles numbers, strings, cell refs, ranges, function calls, and parenthesized exprs.
func (p *parser) parsePrimary() (exprNode, error) {
	t := p.peek()

	switch t.kind {
	case tokNumber:
		p.advance()
		return numberLit{val: t.num}, nil

	case tokString:
		p.advance()
		return stringLit{val: t.text}, nil

	case tokRange:
		p.advance()
		parts := strings.SplitN(t.text, ":", 2)
		return rangeExpr{from: parts[0], to: parts[1]}, nil

	case tokCellRef:
		p.advance()
		return cellRefExpr{ref: t.text}, nil

	case tokFunc:
		return p.parseFuncCall()

	case tokLParen:
		p.advance()
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(tokRParen); err != nil {
			return nil, fmt.Errorf("expected closing parenthesis")
		}
		return expr, nil

	default:
		return nil, fmt.Errorf("unexpected token: %s", t.text)
	}
}

func (p *parser) parseFuncCall() (exprNode, error) {
	name := p.advance() // function name
	if _, err := p.expect(tokLParen); err != nil {
		return nil, fmt.Errorf("expected '(' after function name %s", name.text)
	}

	var args []exprNode
	if p.peek().kind != tokRParen {
		arg, err := p.parseFuncArg()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)

		for p.peek().kind == tokComma {
			p.advance()
			arg, err := p.parseFuncArg()
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
		}
	}

	if _, err := p.expect(tokRParen); err != nil {
		return nil, fmt.Errorf("expected ')' after function arguments")
	}

	return funcCall{name: name.text, args: args}, nil
}

func (p *parser) parseFuncArg() (exprNode, error) {
	// Check for range first (A1:B5)
	if p.peek().kind == tokRange {
		t := p.advance()
		parts := strings.SplitN(t.text, ":", 2)
		return rangeExpr{from: parts[0], to: parts[1]}, nil
	}
	return p.parseExpr()
}

// --- Evaluator ---

// cellLookup is the function signature for looking up a cell value.
type cellLookup func(ref string) (any, bool)

func evaluate(node exprNode, lookup cellLookup) any {
	switch n := node.(type) {
	case numberLit:
		return n.val

	case stringLit:
		return n.val

	case cellRefExpr:
		v, ok := lookup(n.ref)
		if !ok {
			return "#REF!"
		}
		if v == nil {
			return 0.0
		}
		return v

	case rangeExpr:
		return expandRange(n.from, n.to, lookup)

	case unaryExpr:
		val := evaluate(n.operand, lookup)
		if isError(val) {
			return val
		}
		f, ok := asFloat(val)
		if !ok {
			return "#ERR!"
		}
		return -f

	case binaryExpr:
		left := evaluate(n.left, lookup)
		if isError(left) {
			return left
		}
		right := evaluate(n.right, lookup)
		if isError(right) {
			return right
		}

		lf, lok := asFloat(left)
		rf, rok := asFloat(right)

		// String concatenation with +
		if n.op == tokPlus && (!lok || !rok) {
			return fmt.Sprintf("%v%v", left, right)
		}

		if !lok || !rok {
			return "#ERR!"
		}

		switch n.op {
		case tokPlus:
			return lf + rf
		case tokMinus:
			return lf - rf
		case tokStar:
			return lf * rf
		case tokSlash:
			if rf == 0 {
				return "#DIV/0!"
			}
			return lf / rf
		case tokPercent:
			if rf == 0 {
				return "#DIV/0!"
			}
			return math.Mod(lf, rf)
		case tokCaret:
			return math.Pow(lf, rf)
		default:
			return "#ERR!"
		}

	case funcCall:
		return evalFunc(n.name, n.args, lookup)

	default:
		return "#ERR!"
	}
}

func evalFunc(name string, args []exprNode, lookup cellLookup) any {
	// Collect all values (expanding ranges)
	var values []float64
	for _, arg := range args {
		val := evaluate(arg, lookup)
		if isError(val) {
			return val
		}
		switch v := val.(type) {
		case float64:
			values = append(values, v)
		case []any:
			for _, item := range v {
				if isError(item) {
					return item
				}
				f, ok := asFloat(item)
				if !ok {
					continue // skip non-numeric in ranges
				}
				values = append(values, f)
			}
		default:
			f, ok := asFloat(val)
			if ok {
				values = append(values, f)
			}
		}
	}

	switch name {
	case "SUM":
		sum := 0.0
		for _, v := range values {
			sum += v
		}
		return sum

	case "AVG", "AVERAGE":
		if len(values) == 0 {
			return "#DIV/0!"
		}
		sum := 0.0
		for _, v := range values {
			sum += v
		}
		return sum / float64(len(values))

	case "MIN":
		if len(values) == 0 {
			return 0.0
		}
		m := values[0]
		for _, v := range values[1:] {
			if v < m {
				m = v
			}
		}
		return m

	case "MAX":
		if len(values) == 0 {
			return 0.0
		}
		m := values[0]
		for _, v := range values[1:] {
			if v > m {
				m = v
			}
		}
		return m

	case "COUNT":
		return float64(len(values))

	case "ABS":
		if len(values) != 1 {
			return "#ERR!"
		}
		return math.Abs(values[0])

	case "ROUND":
		if len(values) < 1 || len(values) > 2 {
			return "#ERR!"
		}
		places := 0.0
		if len(values) == 2 {
			places = values[1]
		}
		pow := math.Pow(10, places)
		return math.Round(values[0]*pow) / pow

	default:
		return "#ERR!"
	}
}

// expandRange returns a []any of cell values for a range like A1:B5.
func expandRange(from, to string, lookup cellLookup) any {
	fromCol, fromRow := parseCellRef(from)
	toCol, toRow := parseCellRef(to)
	if fromCol == "" || toCol == "" {
		return "#REF!"
	}

	// Normalize so from <= to
	if fromCol > toCol {
		fromCol, toCol = toCol, fromCol
	}
	if fromRow > toRow {
		fromRow, toRow = toRow, fromRow
	}

	var values []any
	for col := fromCol; col <= toCol; col = nextCol(col) {
		for row := fromRow; row <= toRow; row++ {
			ref := fmt.Sprintf("%s%d", col, row)
			v, ok := lookup(ref)
			if !ok {
				values = append(values, 0.0)
			} else if v == nil {
				values = append(values, 0.0)
			} else {
				values = append(values, v)
			}
		}
		if col == toCol {
			break
		}
	}
	return values
}

// parseCellRef splits "A1" into ("A", 1).
func parseCellRef(ref string) (string, int) {
	ref = strings.ToUpper(ref)
	i := 0
	for i < len(ref) && ref[i] >= 'A' && ref[i] <= 'Z' {
		i++
	}
	if i == 0 || i == len(ref) {
		return "", 0
	}
	col := ref[:i]
	row, err := strconv.Atoi(ref[i:])
	if err != nil {
		return "", 0
	}
	return col, row
}

// cellRefString builds a ref string from column and row.
func cellRefString(col string, row int) string {
	return fmt.Sprintf("%s%d", col, row)
}

// nextCol increments a column letter: A->B, Z->AA, etc.
func nextCol(col string) string {
	runes := []rune(col)
	for i := len(runes) - 1; i >= 0; i-- {
		if runes[i] < 'Z' {
			runes[i]++
			return string(runes)
		}
		runes[i] = 'A'
	}
	return "A" + string(runes)
}

// colLetterToIndex converts a column letter to 0-based index (A=0, B=1, ..., Z=25, AA=26).
func colLetterToIndex(col string) int {
	col = strings.ToUpper(col)
	idx := 0
	for _, ch := range col {
		idx = idx*26 + int(ch-'A') + 1
	}
	return idx - 1
}

// indexToColLetter converts 0-based index to column letter.
func indexToColLetter(idx int) string {
	var result []byte
	idx++ // 1-based
	for idx > 0 {
		idx--
		result = append([]byte{byte('A' + idx%26)}, result...)
		idx /= 26
	}
	return string(result)
}

// --- Formula Reference Adjustment ---

// adjustFormula rewrites cell references in a formula by the given row/column deltas.
// Non-formula strings (not starting with '=') are returned as-is.
// References that would become invalid (negative row/col) become #REF!.
func adjustFormula(raw string, rowDelta, colDelta int) string {
	if !strings.HasPrefix(raw, "=") {
		return raw
	}
	formula := raw[1:]
	tokens, err := tokenize(formula)
	if err != nil {
		return raw // unparseable, return as-is
	}

	var b strings.Builder
	b.WriteByte('=')
	for _, tok := range tokens {
		switch tok.kind {
		case tokCellRef:
			b.WriteString(adjustCellRef(tok.text, rowDelta, colDelta))
		case tokRange:
			parts := strings.SplitN(tok.text, ":", 2)
			b.WriteString(adjustCellRef(parts[0], rowDelta, colDelta))
			b.WriteByte(':')
			b.WriteString(adjustCellRef(parts[1], rowDelta, colDelta))
		case tokEOF:
			// skip
		default:
			b.WriteString(tok.text)
		}
	}
	return b.String()
}

// adjustCellRef shifts a single cell reference (e.g. "A1") by rowDelta and colDelta.
func adjustCellRef(ref string, rowDelta, colDelta int) string {
	col, row := parseCellRef(ref)
	if col == "" || row == 0 {
		return ref
	}
	newColIdx := colLetterToIndex(col) + colDelta
	newRow := row + rowDelta
	if newColIdx < 0 || newRow < 1 {
		return "#REF!"
	}
	return indexToColLetter(newColIdx) + strconv.Itoa(newRow)
}

// --- Dependency Graph ---

// DepGraph tracks cell dependencies for formula recalculation.
type DepGraph struct {
	// deps maps a cell ref to the cells it depends on
	deps map[string]map[string]bool
	// rdeps maps a cell ref to cells that depend on it (reverse)
	rdeps map[string]map[string]bool
}

func NewDepGraph() DepGraph {
	return DepGraph{
		deps:  make(map[string]map[string]bool),
		rdeps: make(map[string]map[string]bool),
	}
}

// SetDeps updates the dependencies for a cell.
func (g *DepGraph) SetDeps(cell string, dependsOn []string) {
	cell = strings.ToUpper(cell)

	// Remove old reverse deps
	if old, ok := g.deps[cell]; ok {
		for dep := range old {
			if s, ok := g.rdeps[dep]; ok {
				delete(s, cell)
				if len(s) == 0 {
					delete(g.rdeps, dep)
				}
			}
		}
	}

	// Set new deps
	if len(dependsOn) == 0 {
		delete(g.deps, cell)
		return
	}

	newDeps := make(map[string]bool, len(dependsOn))
	for _, d := range dependsOn {
		d = strings.ToUpper(d)
		newDeps[d] = true

		if g.rdeps[d] == nil {
			g.rdeps[d] = make(map[string]bool)
		}
		g.rdeps[d][cell] = true
	}
	g.deps[cell] = newDeps
}

// Dependents returns all transitive cells that depend on the given cell.
func (g *DepGraph) Dependents(cell string) []string {
	cell = strings.ToUpper(cell)
	visited := make(map[string]bool)
	var result []string
	g.collectDependents(cell, visited, &result)
	return result
}

func (g *DepGraph) collectDependents(cell string, visited map[string]bool, result *[]string) {
	rdeps, ok := g.rdeps[cell]
	if !ok {
		return
	}
	for dep := range rdeps {
		if visited[dep] {
			continue
		}
		visited[dep] = true
		*result = append(*result, dep)
		g.collectDependents(dep, visited, result)
	}
}

// TopoSort returns a topological ordering of the given cells based on their dependencies.
// Returns the sorted list and any cells involved in cycles.
func (g *DepGraph) TopoSort(cells []string) (sorted []string, cycled []string) {
	inSet := make(map[string]bool, len(cells))
	for _, c := range cells {
		inSet[strings.ToUpper(c)] = true
	}

	// Compute in-degree for cells within the set
	inDegree := make(map[string]int)
	for _, c := range cells {
		c = strings.ToUpper(c)
		inDegree[c] = 0
	}
	for _, c := range cells {
		c = strings.ToUpper(c)
		if deps, ok := g.deps[c]; ok {
			for dep := range deps {
				if inSet[dep] {
					inDegree[c]++
				}
			}
		}
	}

	// Kahn's algorithm
	var queue []string
	for _, c := range cells {
		c = strings.ToUpper(c)
		if inDegree[c] == 0 {
			queue = append(queue, c)
		}
	}

	for len(queue) > 0 {
		cell := queue[0]
		queue = queue[1:]
		sorted = append(sorted, cell)

		if rdeps, ok := g.rdeps[cell]; ok {
			for dep := range rdeps {
				if !inSet[dep] {
					continue
				}
				inDegree[dep]--
				if inDegree[dep] == 0 {
					queue = append(queue, dep)
				}
			}
		}
	}

	// Any remaining cells with non-zero in-degree are in a cycle
	for _, c := range cells {
		c = strings.ToUpper(c)
		found := false
		for _, s := range sorted {
			if s == c {
				found = true
				break
			}
		}
		if !found {
			cycled = append(cycled, c)
		}
	}

	return sorted, cycled
}

// extractDeps extracts all cell references from an AST node.
func extractDeps(node exprNode) []string {
	var deps []string
	extractDepsHelper(node, &deps)
	return deps
}

func extractDepsHelper(node exprNode, deps *[]string) {
	switch n := node.(type) {
	case cellRefExpr:
		*deps = append(*deps, strings.ToUpper(n.ref))
	case rangeExpr:
		// Add all cells in the range
		fromCol, fromRow := parseCellRef(n.from)
		toCol, toRow := parseCellRef(n.to)
		if fromCol > toCol {
			fromCol, toCol = toCol, fromCol
		}
		if fromRow > toRow {
			fromRow, toRow = toRow, fromRow
		}
		for col := fromCol; col <= toCol; col = nextCol(col) {
			for row := fromRow; row <= toRow; row++ {
				*deps = append(*deps, cellRefString(col, row))
			}
			if col == toCol {
				break
			}
		}
	case binaryExpr:
		extractDepsHelper(n.left, deps)
		extractDepsHelper(n.right, deps)
	case unaryExpr:
		extractDepsHelper(n.operand, deps)
	case funcCall:
		for _, arg := range n.args {
			extractDepsHelper(arg, deps)
		}
	}
}

// isError checks if a value is an error string.
func isError(v any) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	return strings.HasPrefix(s, "#")
}

// asFloat attempts to convert a value to float64.
func asFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case string:
		f, err := strconv.ParseFloat(n, 64)
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}

// --- Recalculation ---

// recalcAll parses all formulas, builds the dependency graph, and evaluates in order.
func recalcAll(rows []*SpreadsheetRow, deps *DepGraph) {
	*deps = NewDepGraph()

	// Collect all cells with formulas
	var allCells []string
	for _, row := range rows {
		for col, cell := range row.Cells {
			ref := cellRefString(col, row.RowIndex+1) // 1-based row
			if strings.HasPrefix(cell.Raw, "=") {
				ast, err := parse(cell.Raw[1:])
				if err != nil {
					cell.Value = "#ERR!"
					continue
				}
				cellDeps := extractDeps(ast)
				deps.SetDeps(ref, cellDeps)
				allCells = append(allCells, ref)
			} else {
				deps.SetDeps(ref, nil)
				// Parse literal value
				cell.Value = parseLiteral(cell.Raw)
			}
		}
	}

	// Topological sort
	sorted, cycled := deps.TopoSort(allCells)

	// Mark cycles
	for _, ref := range cycled {
		col, rowNum := parseCellRef(ref)
		if c := getCell(rows, col, rowNum); c != nil {
			c.Value = "#CIRC!"
		}
	}

	// Evaluate in topological order
	lookup := makeLookup(rows)
	for _, ref := range sorted {
		col, rowNum := parseCellRef(ref)
		c := getCell(rows, col, rowNum)
		if c == nil {
			continue
		}
		if !strings.HasPrefix(c.Raw, "=") {
			continue
		}
		ast, err := parse(c.Raw[1:])
		if err != nil {
			c.Value = "#ERR!"
			continue
		}
		c.Value = evaluate(ast, lookup)
	}
}

// recalcFrom incrementally recalculates starting from a changed cell.
func recalcFrom(ref string, rows []*SpreadsheetRow, deps *DepGraph) {
	ref = strings.ToUpper(ref)

	// First, update the changed cell itself
	col, rowNum := parseCellRef(ref)
	c := getCell(rows, col, rowNum)
	if c != nil {
		if strings.HasPrefix(c.Raw, "=") {
			ast, err := parse(c.Raw[1:])
			if err != nil {
				c.Value = "#ERR!"
				deps.SetDeps(ref, nil)
			} else {
				cellDeps := extractDeps(ast)
				deps.SetDeps(ref, cellDeps)
			}
		} else {
			c.Value = parseLiteral(c.Raw)
			deps.SetDeps(ref, nil)
		}
	}

	// Find transitive dependents
	dependents := deps.Dependents(ref)
	if len(dependents) == 0 {
		// Still need to evaluate the changed cell if it's a formula
		if c != nil && strings.HasPrefix(c.Raw, "=") {
			lookup := makeLookup(rows)
			ast, err := parse(c.Raw[1:])
			if err == nil {
				c.Value = evaluate(ast, lookup)
			}
		}
		return
	}

	// Include the changed cell itself in the evaluation set
	allCells := append([]string{ref}, dependents...)
	sorted, cycled := deps.TopoSort(allCells)

	// Mark cycles
	for _, cycRef := range cycled {
		cc, cr := parseCellRef(cycRef)
		if cell := getCell(rows, cc, cr); cell != nil {
			cell.Value = "#CIRC!"
		}
	}

	// Evaluate in order
	lookup := makeLookup(rows)
	for _, evalRef := range sorted {
		ec, er := parseCellRef(evalRef)
		cell := getCell(rows, ec, er)
		if cell == nil || !strings.HasPrefix(cell.Raw, "=") {
			continue
		}
		ast, err := parse(cell.Raw[1:])
		if err != nil {
			cell.Value = "#ERR!"
			continue
		}
		cell.Value = evaluate(ast, lookup)
	}
}

// getCell finds a cell by column letter and 1-based row number.
func getCell(rows []*SpreadsheetRow, col string, rowNum int) *Cell {
	rowIdx := rowNum - 1
	if rowIdx < 0 || rowIdx >= len(rows) {
		return nil
	}
	return rows[rowIdx].Cells[col]
}

// makeLookup creates a cellLookup function from the spreadsheet rows.
func makeLookup(rows []*SpreadsheetRow) cellLookup {
	return func(ref string) (any, bool) {
		col, rowNum := parseCellRef(ref)
		if col == "" {
			return nil, false
		}
		c := getCell(rows, col, rowNum)
		if c == nil {
			return 0.0, true // empty cell = 0
		}
		return c.Value, true
	}
}

// parseLiteral converts a raw string to a typed value.
func parseLiteral(raw string) any {
	if raw == "" {
		return nil
	}
	// Try float
	if f, err := strconv.ParseFloat(raw, 64); err == nil {
		return f
	}
	// Otherwise it's a string
	return raw
}
