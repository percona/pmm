package main

import (
	"go/ast"
	"go/token"
	"go/types"
	"regexp"
	"strings"

	"github.com/go-toolsmith/astcast"
	"github.com/go-toolsmith/astequal"
	"github.com/go-toolsmith/typep"
)

func (ctxt *context) mark(n ast.Node, v *opVariant) {
	v.count++
	pos := ctxt.fset.Position(n.Pos())
	ctxt.candidates = append(ctxt.candidates, candidate{
		variantID:  v.id,
		locationID: ctxt.locs.Insert(pos.Filename, pos.Line, pos.Column),
	})
}

type operation struct {
	// name is a human-readable operation descriptor.
	//
	// Initialized by checker constructor.
	name string

	// suggested is an op variant that is inferred as the most frequently used one.
	//
	// Updated during the context.assignSuggestions.
	suggested *opVariant

	// variants is a list of equivalent operation forms.
	//
	// Initialized by checker constructor.
	variants []*opVariant
}

type opVariant struct {
	// id is an globally-unique operation variant ID.
	//
	// Initialized by context.initCheckers.
	id int

	// op is a reference to a containing operation.
	//
	// Initialized by context.initCheckers.
	op *operation

	// warning is a message to use if this variant is not used
	// when it is the suggested one.
	//
	// For example, if the variant requires the usage of the
	// parenthesis, warning should suggest adding them.
	//
	// Initialized by checker constructor.
	warning string

	// count is a counter for op variant usages.
	//
	// Updated during the context.collectCandidates.
	count int
}

type checker interface {
	Visit(n ast.Node) bool
	Operation() *operation
}

type checkerBase struct {
	ctxt *context
	op   *operation
}

func (c *checkerBase) Operation() *operation {
	return c.op
}

type candidate struct {
	variantID  int
	locationID int
}

type defaultCaseOrderChecker struct {
	checkerBase

	first opVariant
	last  opVariant
}

func newDefaultCaseOrderChecker(ctxt *context) checker {
	c := &defaultCaseOrderChecker{}
	c.ctxt = ctxt
	c.first.warning = "default case should be the first case"
	c.last.warning = "default case should be the last case"
	c.op = &operation{
		name:     "default case order",
		variants: []*opVariant{&c.first, &c.last},
	}
	return c
}

func (c *defaultCaseOrderChecker) Visit(n ast.Node) bool {
	cases := c.casesList(n)
	if len(cases) < 2 {
		return true
	}
	switch c.defaultCaseIndex(cases) {
	case 0:
		c.ctxt.mark(n, &c.first)
	case len(cases) - 1:
		c.ctxt.mark(n, &c.last)
	}
	return true
}

func (c *defaultCaseOrderChecker) casesList(n ast.Node) []ast.Stmt {
	switch n := n.(type) {
	case *ast.TypeSwitchStmt:
		return n.Body.List
	case *ast.SwitchStmt:
		return n.Body.List
	default:
		return nil
	}
}

func (c *defaultCaseOrderChecker) defaultCaseIndex(cases []ast.Stmt) int {
	for i, stmt := range cases {
		cc := stmt.(*ast.CaseClause)
		if cc.List == nil {
			return i
		}
	}
	return -1
}

type nonZeroLenTestChecker struct {
	checkerBase

	neq0 opVariant
	gt0  opVariant
	gte1 opVariant
}

func newNonZeroLenTestChecker(ctxt *context) checker {
	c := &nonZeroLenTestChecker{}
	c.ctxt = ctxt
	c.neq0.warning = "use `len(s) != 0`"
	c.gt0.warning = "use `len(s) > 0`"
	c.gte1.warning = "use `len(s) >= 1`"
	c.op = &operation{
		name:     "non-zero length test",
		variants: []*opVariant{&c.neq0, &c.gt0, &c.gte1},
	}
	return c
}

func (c *nonZeroLenTestChecker) Visit(n ast.Node) bool {
	cmp := astcast.ToBinaryExpr(n)
	call := astcast.ToCallExpr(cmp.X)
	if len(call.Args) != 1 || astcast.ToIdent(call.Fun).Name != "len" {
		return true
	}
	x := call.Args[0]
	if typep.HasStringKind(c.ctxt.info.TypeOf(x)) {
		return true
	}
	switch val := valueOf(cmp.Y); {
	case cmp.Op == token.NEQ && val == "0":
		c.ctxt.mark(n, &c.neq0)
	case cmp.Op == token.GTR && val == "0":
		c.ctxt.mark(n, &c.gt0)
	case cmp.Op == token.GEQ && val == "1":
		c.ctxt.mark(n, &c.gte1)
	}
	return true
}

type zeroValPtrAllocChecker struct {
	checkerBase

	newCall      opVariant
	addressOfLit opVariant
}

func newZeroValPtrAllocChecker(ctxt *context) checker {
	c := &zeroValPtrAllocChecker{}
	c.ctxt = ctxt
	c.newCall.warning = "use new(T) for *T allocation"
	c.addressOfLit.warning = "use &T{} for *T allocation"
	c.op = &operation{
		name:     "zero value ptr alloc",
		variants: []*opVariant{&c.newCall, &c.addressOfLit},
	}
	return c
}

func (c *zeroValPtrAllocChecker) Visit(n ast.Node) bool {
	switch n := n.(type) {
	case *ast.CallExpr:
		fn, ok := n.Fun.(*ast.Ident)
		if !ok || fn.Name != "new" || len(n.Args) != 1 {
			return true
		}
		typ := c.ctxt.info.TypeOf(n.Args[0])
		if _, ok := typ.(*types.Basic); ok {
			return true
		}
		c.ctxt.mark(n, &c.newCall)
	case *ast.UnaryExpr:
		lit, ok := n.X.(*ast.CompositeLit)
		if !ok || n.Op != token.AND || len(lit.Elts) != 0 {
			return true
		}
		typ := c.ctxt.info.TypeOf(lit.Type)
		if _, ok := typ.(*types.Basic); ok {
			return true
		}
		c.ctxt.mark(n, &c.addressOfLit)
	}
	return true
}

type hexLitChecker struct {
	checkerBase

	lowerCase opVariant
	upperCase opVariant
}

func newHexLitChecker(ctxt *context) checker {
	c := &hexLitChecker{}
	c.ctxt = ctxt
	c.lowerCase.warning = "use a-f (lower case) digits"
	c.upperCase.warning = "use A-F (upper case) digits"
	c.op = &operation{
		name:     "hex lit",
		variants: []*opVariant{&c.lowerCase, &c.upperCase},
	}
	return c
}

func (c *hexLitChecker) Visit(n ast.Node) bool {
	lit, ok := n.(*ast.BasicLit)
	if !ok {
		return true
	}
	if lit.Kind != token.INT || !strings.HasPrefix(lit.Value, "0x") {
		return false
	}
	switch {
	case strings.ContainsAny(lit.Value, "abcdef"):
		c.ctxt.mark(n, &c.lowerCase)
	case strings.ContainsAny(lit.Value, "ABCDEF"):
		c.ctxt.mark(n, &c.upperCase)
	}
	return false
}

type rangeCheckChecker struct {
	checkerBase

	alignLeft   opVariant
	alignCenter opVariant
}

func newRangeCheckChecker(ctxt *context) checker {
	c := &rangeCheckChecker{}
	c.ctxt = ctxt
	c.alignLeft.warning = "use align-left, like in `x >= low && x <= high`"
	c.alignCenter.warning = "use align-center, like in `low < x && x < high`"
	c.op = &operation{
		name:     "range check",
		variants: []*opVariant{&c.alignLeft, &c.alignCenter},
	}
	return c
}

func (c *rangeCheckChecker) Visit(n ast.Node) bool {
	e := astcast.ToBinaryExpr(n)
	if e.Op != token.LAND && e.Op != token.LOR {
		return true
	}
	lhs := astcast.ToBinaryExpr(e.X)
	rhs := astcast.ToBinaryExpr(e.Y)

	leftAligned := (lhs.Op == token.GTR || lhs.Op == token.GEQ) &&
		(rhs.Op == token.LSS || rhs.Op == token.LEQ) &&
		astequal.Expr(lhs.X, rhs.X)
	if leftAligned {
		c.ctxt.mark(n, &c.alignLeft)
		return false
	}

	centerAligned := (lhs.Op == token.LSS || lhs.Op == token.LEQ) &&
		(rhs.Op == token.LSS || lhs.Op == token.LEQ) &&
		astequal.Expr(lhs.Y, rhs.X)
	if centerAligned {
		c.ctxt.mark(n, &c.alignCenter)
		return false
	}

	return true
}

type andNotChecker struct {
	checkerBase

	noSpace   opVariant
	withSpace opVariant
}

func newAndNotChecker(ctxt *context) checker {
	c := &andNotChecker{}
	c.ctxt = ctxt
	c.noSpace.warning = "remove a space between & and ^, like in `x &^ y`"
	c.withSpace.warning = "put a space between & and ^, like in `x & ^y`"
	c.op = &operation{
		name:     "and-not",
		variants: []*opVariant{&c.noSpace, &c.withSpace},
	}
	return c
}

func (c *andNotChecker) Visit(n ast.Node) bool {
	e := astcast.ToBinaryExpr(n)
	switch {
	case e.Op == token.AND_NOT:
		c.ctxt.mark(n, &c.noSpace)
	case e.Op == token.AND && astcast.ToUnaryExpr(e.Y).Op == token.XOR:
		c.ctxt.mark(n, &c.withSpace)
	}
	return true
}

type floatLitChecker struct {
	checkerBase

	explicitIntFrac opVariant
	implicitIntFrac opVariant
}

func newFloatLitChecker(ctxt *context) checker {
	c := &floatLitChecker{}
	c.ctxt = ctxt
	c.explicitIntFrac.warning = "use explicit int/frac part, like in `1.0` and `0.1`"
	c.implicitIntFrac.warning = "use implicit int/frac part, like in `1.` and `.1`"
	c.op = &operation{
		name:     "float lit",
		variants: []*opVariant{&c.explicitIntFrac, &c.implicitIntFrac},
	}
	return c
}

func (c *floatLitChecker) Visit(n ast.Node) bool {
	lit, ok := n.(*ast.BasicLit)
	if !ok {
		return true
	}
	if lit.Kind != token.FLOAT {
		return false
	}
	integer, frac := c.splitIntFrac(lit)
	switch {
	case (integer == "0" && frac != "") || (integer != "" && frac == "0"):
		c.ctxt.mark(n, &c.explicitIntFrac)
	case (integer != "" && frac == "") || (integer == "" && frac != ""):
		c.ctxt.mark(n, &c.implicitIntFrac)
	}
	return false
}

func (c *floatLitChecker) splitIntFrac(n *ast.BasicLit) (integer, frac string) {
	parts := strings.Split(n.Value, ".")
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

type labelCaseChecker struct {
	checkerBase

	allUpperCase   opVariant
	upperCamelCase opVariant
	lowerCamelCase opVariant

	allUpperCaseRE   *regexp.Regexp
	upperCamelCaseRE *regexp.Regexp
	lowerCamelCaseRE *regexp.Regexp
}

func newLabelCaseChecker(ctxt *context) checker {
	c := &labelCaseChecker{}
	c.ctxt = ctxt
	c.allUpperCase.warning = "use ALL_UPPER"
	c.upperCamelCase.warning = "use UpperCamelCase"
	c.lowerCamelCase.warning = "use lowerCamelCase"
	c.allUpperCaseRE = regexp.MustCompile(`^[A-Z][A-Z_0-9]*$`)
	c.upperCamelCaseRE = regexp.MustCompile(`^[A-Z]\w*$`)
	c.lowerCamelCaseRE = regexp.MustCompile(`^[a-z]\w*$`)
	c.op = &operation{
		name: "label case",
		variants: []*opVariant{
			&c.allUpperCase,
			&c.upperCamelCase,
			&c.lowerCamelCase,
		},
	}
	return c
}

func (c *labelCaseChecker) Visit(n ast.Node) bool {
	stmt, ok := n.(*ast.LabeledStmt)
	if !ok {
		return true
	}
	switch {
	case c.allUpperCaseRE.MatchString(stmt.Label.Name):
		c.ctxt.mark(n, &c.allUpperCase)
	case c.upperCamelCaseRE.MatchString(stmt.Label.Name):
		c.ctxt.mark(n, &c.upperCamelCase)
	case c.lowerCamelCaseRE.MatchString(stmt.Label.Name):
		c.ctxt.mark(n, &c.lowerCamelCase)
	}
	return true
}

type untypedConstCoerceChecker struct {
	checkerBase

	lhsType opVariant
	rhsType opVariant
}

func newUntypedConstCoerceChecker(ctxt *context) checker {
	c := &untypedConstCoerceChecker{}
	c.ctxt = ctxt
	c.lhsType.warning = "specify type in LHS, like in `var x T = const`"
	c.rhsType.warning = "specity type in RHS, like in `var x = T(const)`"
	c.op = &operation{
		name:     "untyped const coerce",
		variants: []*opVariant{&c.lhsType, &c.rhsType},
	}
	return c
}

func (c *untypedConstCoerceChecker) Visit(n ast.Node) bool {
	decl, ok := n.(*ast.GenDecl)
	if !ok {
		return true
	}
	if decl.Tok != token.VAR && decl.Tok != token.CONST {
		return false
	}
	if len(decl.Specs) != 1 {
		return false
	}
	spec := decl.Specs[0].(*ast.ValueSpec)
	if len(spec.Names) != 1 || len(spec.Values) != 1 {
		return false
	}

	if spec.Type != nil {
		if !c.isUntypedConst(spec.Values[0]) {
			return false
		}
		c.ctxt.mark(n, &c.lhsType)
	} else {
		conv, ok := spec.Values[0].(*ast.CallExpr)
		if !ok {
			return false
		}
		if len(conv.Args) != 1 || !c.isUntypedConst(conv.Args[0]) {
			return false
		}
		c.ctxt.mark(n, &c.rhsType)
	}

	return false
}

func (c *untypedConstCoerceChecker) isUntypedConst(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.BasicLit:
		return true
	case *ast.Ident:
		typ, ok := c.ctxt.info.ObjectOf(e).Type().(*types.Basic)
		return ok && typ.Info()&types.IsUntyped != 0
	case *ast.BinaryExpr:
		return c.isUntypedConst(e.X) && c.isUntypedConst(e.Y)
	case *ast.UnaryExpr:
		return c.isUntypedConst(e.X)
	case *ast.ParenExpr:
		return c.isUntypedConst(e.X)
	default:
		return false
	}
}

type emptyMapChecker struct {
	checkerBase

	makeCall opVariant
	mapLit   opVariant
}

func newEmptyMapChecker(ctxt *context) checker {
	c := &emptyMapChecker{}
	c.ctxt = ctxt
	c.makeCall.warning = "use make(map[K]V)"
	c.mapLit.warning = "use map[K]V{}"
	c.op = &operation{
		name:     "empty map",
		variants: []*opVariant{&c.makeCall, &c.mapLit},
	}
	return c
}

func (c *emptyMapChecker) Visit(n ast.Node) bool {
	switch n := n.(type) {
	case *ast.CallExpr:
		fn, ok := n.Fun.(*ast.Ident)
		if !ok || fn.Name != "make" {
			return true
		}
		typ := c.ctxt.info.TypeOf(n.Args[0])
		if _, ok := typ.(*types.Map); !ok {
			return true
		}
		if len(n.Args) == 2 && valueOf(n.Args[1]) != "0" {
			return true
		}
		c.ctxt.mark(n, &c.makeCall)
	case *ast.CompositeLit:
		// Avoid &map[K]V{}, since it's a new(map[K]V), not make(map[K]V).
		unExpr, ok := c.ctxt.astinfo.Parents[n].(*ast.UnaryExpr)
		if ok && unExpr.Op == token.AND {
			return true
		}
		typ := c.ctxt.info.TypeOf(n.Type)
		if _, ok := typ.(*types.Map); !ok {
			return true
		}
		if len(n.Elts) != 0 {
			return true
		}
		c.ctxt.mark(n, &c.mapLit)
	}
	return true
}

type emptySliceChecker struct {
	checkerBase

	makeCall opVariant
	sliceLit opVariant
}

func newEmptySliceChecker(ctxt *context) checker {
	c := &emptySliceChecker{}
	c.ctxt = ctxt
	c.makeCall.warning = "use make([]T, 0)"
	c.sliceLit.warning = "use []T{}"
	c.op = &operation{
		name:     "empty slice",
		variants: []*opVariant{&c.makeCall, &c.sliceLit},
	}
	return c
}

func (c *emptySliceChecker) Visit(n ast.Node) bool {
	switch n := n.(type) {
	case *ast.CallExpr:
		fn, ok := n.Fun.(*ast.Ident)
		if !ok || fn.Name != "make" || len(n.Args) != 2 {
			return true
		}
		typ := c.ctxt.info.TypeOf(n.Args[0])
		if _, ok := typ.(*types.Slice); !ok {
			return true
		}
		if valueOf(n.Args[1]) != "0" {
			return true
		}
		c.ctxt.mark(n, &c.makeCall)
	case *ast.CompositeLit:
		// Avoid &[]T{}, since it's a new([]T), not make([]T, 0).
		unExpr, ok := c.ctxt.astinfo.Parents[n].(*ast.UnaryExpr)
		if ok && unExpr.Op == token.AND {
			return true
		}
		typ := c.ctxt.info.TypeOf(n.Type)
		if _, ok := typ.(*types.Slice); !ok {
			return true
		}
		if len(n.Elts) != 0 {
			return true
		}
		c.ctxt.mark(n, &c.sliceLit)
	}
	return true
}

type argListParensChecker struct {
	checkerBase

	sameLine opVariant
	nextLine opVariant
}

func newArgListParensChecker(ctxt *context) checker {
	c := &argListParensChecker{}
	c.ctxt = ctxt
	c.sameLine.warning = "align `)` to a same line with last argument"
	c.nextLine.warning = "move `)` to the next line and put `,` after the last argument"
	c.op = &operation{
		name:     "arg list parens",
		variants: []*opVariant{&c.sameLine, &c.nextLine},
	}
	return c
}

func (c *argListParensChecker) Visit(n ast.Node) bool {
	call, ok := n.(*ast.CallExpr)
	if !ok || len(call.Args) < 2 {
		return true
	}
	lastArg := call.Args[len(call.Args)-1]
	lastArgLine := c.ctxt.fset.Position(lastArg.Pos()).Line
	firstArgLine := c.ctxt.fset.Position(call.Args[0].Pos()).Line
	if firstArgLine == lastArgLine {
		// Don't track single-line function calls.
		return true
	}
	rparenLine := c.ctxt.fset.Position(call.Rparen).Line
	switch rparenLine {
	case lastArgLine:
		c.ctxt.mark(n, &c.sameLine)
	case lastArgLine + 1:
		c.ctxt.mark(n, &c.nextLine)
	}
	return true
}

type unitImportChecker struct {
	checkerBase

	noParens   opVariant
	withParens opVariant
}

func newUnitImportChecker(ctxt *context) checker {
	c := &unitImportChecker{}
	c.ctxt = ctxt
	c.noParens.warning = "omit parenthesis in a single-package import"
	c.withParens.warning = "wrap single-package import spec into parenthesis"
	c.op = &operation{
		name:     "unit import",
		variants: []*opVariant{&c.noParens, &c.withParens},
	}
	return c
}

func (c *unitImportChecker) Visit(n ast.Node) bool {
	decl, ok := n.(*ast.GenDecl)
	if ok && decl.Tok == token.IMPORT && len(decl.Specs) == 1 {
		if decl.Lparen == 0 && decl.Rparen == 0 {
			c.ctxt.mark(n, &c.noParens)
		} else {
			c.ctxt.mark(n, &c.withParens)
		}
	}
	return false
}
