package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/token"
	"go/types"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/go-toolsmith/astinfo"
	"github.com/go-toolsmith/pkgload"
	"github.com/kisielk/gotool"
	"golang.org/x/tools/go/packages"
)

var generatedFileCommentRE = regexp.MustCompile("Code generated .* DO NOT EDIT.")

func main() {
	log.SetFlags(0)
	var ctxt context

	steps := []struct {
		name string
		fn   func() error
	}{
		{"parse flags", ctxt.parseFlags},
		{"resolve targets", ctxt.resolveTargets},
		{"init checkers", ctxt.initCheckers},
		{"collect candidates", ctxt.collectAllCandidates},
		{"assign suggestions", ctxt.assignSuggestions},
		{"print warnings", ctxt.printWarnings},
	}

	for _, step := range steps {
		if err := step.fn(); err != nil {
			log.Fatalf("%s: %v", step.name, err)
		}
	}
}

type context struct {
	// flags is an (effectively) immutable struct that holds all command-line
	// arguments as they were passed to the program.
	//
	// For per-argument documentation see context.parseFlags.
	flags struct {
		pedantic           bool
		verbose            bool
		shorterErrLocation bool

		targets []string
		exclude string
	}

	workDir string

	paths []string

	locs *locationMap

	fset    *token.FileSet
	info    *types.Info
	astinfo astinfo.Info

	checkers []checker

	candidates []candidate
}

func (ctxt *context) parseFlags() error {
	flag.BoolVar(&ctxt.flags.pedantic, "pedantic", false,
		`makes several diagnostics more pedantic and comprehensive`)
	flag.BoolVar(&ctxt.flags.verbose, "v", false,
		`turn on detailed program execution info printing`)
	flag.BoolVar(&ctxt.flags.shorterErrLocation, `shorterErrLocation`, true,
		`whether to replace error location prefix with $GOROOT and $GOPATH`)
	flag.StringVar(&ctxt.flags.exclude, "exclude", `^unsafe$|^builtin$`,
		`import path excluding regexp`)

	flag.Parse()

	ctxt.flags.targets = flag.Args()
	if len(ctxt.flags.targets) == 0 {
		return fmt.Errorf("not enough positional args (empty targets list)")
	}

	if ctxt.flags.shorterErrLocation {
		wd, err := os.Getwd()
		if err != nil {
			log.Printf("getwd: %v", err)
		}
		ctxt.workDir = wd
	}

	return nil
}

func (ctxt *context) resolveTargets() error {
	ctxt.paths = gotool.ImportPaths(ctxt.flags.targets)
	if len(ctxt.paths) == 0 {
		return fmt.Errorf("targets resolved to an empty import paths list")
	}

	// Filter-out packages using the exclude pattern.
	excludeRE, err := regexp.Compile(ctxt.flags.exclude)
	if err != nil {
		return fmt.Errorf("compiling -exclude regexp: %v", err)
	}
	paths := ctxt.paths[:0]
	for _, path := range ctxt.paths {
		if !excludeRE.MatchString(path) {
			paths = append(paths, path)
		}
	}
	ctxt.paths = paths

	if len(paths) == 0 {
		ctxt.infoPrintf("import paths list is empty after filtering")
	}

	return nil
}

func (ctxt *context) initCheckers() error {
	checkers := []checker{
		newUnitImportChecker(ctxt),
		newZeroValPtrAllocChecker(ctxt),
		newEmptySliceChecker(ctxt),
		newEmptyMapChecker(ctxt),
		newHexLitChecker(ctxt),
		newRangeCheckChecker(ctxt),
		newAndNotChecker(ctxt),
		newFloatLitChecker(ctxt),
		newLabelCaseChecker(ctxt),
		newUntypedConstCoerceChecker(ctxt),
		newArgListParensChecker(ctxt),
		newNonZeroLenTestChecker(ctxt),
		newDefaultCaseOrderChecker(ctxt),
	}

	variantID := 0
	for _, c := range checkers {
		op := c.Operation()
		if op.name == "" {
			panic(fmt.Sprintf("%T: empty operation name", c))
		}
		for i, v := range op.variants {
			if v.warning == "" {
				panic(fmt.Sprintf("%T: empty warning for variant#%d", c, i))
			}
			v.op = op
			v.id = variantID
			variantID++
		}
	}

	ctxt.locs = newLocationMap()
	ctxt.checkers = checkers

	return nil
}

func (ctxt *context) collectAllCandidates() error {
	for _, path := range ctxt.paths {
		ctxt.infoPrintf("check %q", path)
		if err := ctxt.collectPathCandidates(path); err != nil {
			return fmt.Errorf("%s: %v", path, err)
		}
	}
	return nil
}

func (ctxt *context) collectPackageCandidates(pkg *packages.Package) {
	ctxt.info = pkg.TypesInfo
	for _, f := range pkg.Syntax {
		isGenerated := len(f.Comments) != 0 &&
			generatedFileCommentRE.MatchString(f.Comments[0].Text())
		if isGenerated {
			continue
		}
		ctxt.collectFileCandidates(f)
	}
}

func (ctxt *context) collectPathCandidates(path string) error {
	ctxt.fset = token.NewFileSet()

	conf := &packages.Config{
		Mode:  packages.LoadSyntax,
		Fset:  ctxt.fset,
		Tests: true,
	}

	// TODO(Quasilyte): current approach is memory-efficient
	// and does scale well with huge amounts of targets to check,
	// but it's not very fast. Might want to optimize it a little bit.
	pkgs, err := packages.Load(conf, path)
	if err != nil {
		return err
	}
	if len(pkgs) == 0 {
		ctxt.infoPrintf("got 0 packages for %q path", path)
		return nil
	}
	if n := packages.PrintErrors(pkgs); n > 0 {
		return fmt.Errorf("%d build errors", n)
	}

	pkgload.VisitUnits(pkgs, func(u *pkgload.Unit) {
		if u.ExternalTest != nil {
			ctxt.collectPackageCandidates(u.ExternalTest)
		}
		if u.Test != nil {
			// Prefer tests to the base package, if present.
			ctxt.collectPackageCandidates(u.Test)
		} else {
			ctxt.collectPackageCandidates(u.Base)
		}
	})

	return nil
}

func (ctxt *context) collectFileCandidates(f *ast.File) {
	ctxt.astinfo = astinfo.Info{
		Parents: make(map[ast.Node]ast.Node),
	}
	ctxt.astinfo.Origin = f
	ctxt.astinfo.Resolve()

	for _, c := range ctxt.checkers {
		for _, decl := range f.Decls {
			ast.Inspect(decl, c.Visit)
		}
	}
}

func (ctxt *context) assignSuggestions() error {
	for _, c := range ctxt.checkers {
		op := c.Operation()
		op.suggested = op.variants[0]
		for _, v := range op.variants[1:] {
			if v.count > op.suggested.count {
				op.suggested = v
			}
		}
	}
	return nil
}

func (ctxt *context) printWarnings() error {
	exitCode := 0
	visitWarnings(ctxt, func(pos token.Position, v *opVariant) {
		exitCode = 1
		loc := pos.String()
		if ctxt.flags.shorterErrLocation {
			loc = ctxt.shortenLocation(loc)
		}
		fmt.Printf("%s: %s: %s\n", loc, v.op.name, v.op.suggested.warning)
	})
	os.Exit(exitCode)
	return nil
}

func visitWarnings(ctxt *context, visit func(pos token.Position, v *opVariant)) {
	// Build variant map which is accessed by variantID.
	vcount := 0
	for _, c := range ctxt.checkers {
		vcount += len(c.Operation().variants)
	}
	variants := make([]*opVariant, vcount)
	for _, c := range ctxt.checkers {
		for _, v := range c.Operation().variants {
			variants[v.id] = v
		}
	}

	for _, c := range ctxt.candidates {
		v := variants[c.variantID]
		if v.op.suggested == v {
			continue // OK, everything is consistent
		}
		pos := ctxt.locs.Get(c.locationID)
		visit(pos, v)
	}
}

func (ctxt *context) shortenLocation(loc string) string {
	// If possible, construct relative path.
	relLoc := loc
	if ctxt.workDir != "" {
		relLoc = strings.Replace(loc, ctxt.workDir, ".", 1)
	}

	switch {
	case strings.HasPrefix(loc, build.Default.GOPATH):
		loc = strings.Replace(loc, build.Default.GOPATH, "$GOPATH", 1)
	case strings.HasPrefix(loc, build.Default.GOROOT):
		loc = strings.Replace(loc, build.Default.GOROOT, "$GOROOT", 1)
	}

	// Return the representation that is shorter.
	if len(relLoc) < len(loc) {
		return relLoc
	}
	return loc
}

func (ctxt *context) infoPrintf(format string, args ...interface{}) {
	if ctxt.flags.verbose {
		log.Printf("\tinfo: "+format, args...)
	}
}
