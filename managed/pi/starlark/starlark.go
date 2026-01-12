// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// Package starlark provides Starlark execution environment.
package starlark

import (
	"errors"
	"fmt"

	"go.starlark.net/starlark"

	"github.com/percona/pmm/managed/pi/check"
	"github.com/percona/pmm/managed/pi/common"
)

// PrintFunc represents fmt.Println-like function that is used by Starlark 'print' function implementation.
type PrintFunc func(args ...any)

// GoFunc represent a Go function that can be registered in Starlark environment.
type GoFunc func(args ...any) (any, error)

// Env represents Starlark execution environment.
type Env struct {
	p           *starlark.Program
	predeclared starlark.StringDict
}

// NewEnv creates a new Starlark execution environment.
func NewEnv(name, script string, predeclaredFuncs map[string]GoFunc) (*Env, error) {
	var env *Env

	predeclared := make(starlark.StringDict, len(predeclaredFuncs))

	for n, f := range predeclaredFuncs {
		predeclared[n] = starlark.NewBuiltin(n, makeFunc(f))
	}

	predeclared.Freeze()

	var p *starlark.Program

	_, p, err := starlark.SourceProgram(name, script, predeclared.Has)
	if err != nil {
		return env, err
	}

	env = &Env{
		p:           p,
		predeclared: predeclared,
	}

	return env, err
}

// starlarkFunc represent a Starlark builtin_function_or_method.
type starlarkFunc func(*starlark.Thread, *starlark.Builtin, starlark.Tuple, []starlark.Tuple) (starlark.Value, error)

// makeFunc converts GoFunc to starlarkFunc.
func makeFunc(f GoFunc) starlarkFunc {
	return func(_ *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if len(kwargs) != 0 {
			return nil, fmt.Errorf("%s: kwargs are not supported", fn.Name())
		}

		fargs := make([]any, len(args))
		for i, arg := range args {
			farg, err := starlarkToGo(arg)
			if err != nil {
				return nil, err
			}

			fargs[i] = farg
		}

		res, err := f(fargs...)
		if err != nil {
			return nil, err
		}

		v, err := goToStarlark(res)
		if err != nil {
			return nil, err
		}

		return v, nil
	}
}

// noopPrint is a no-op 'print' implementation.
// It is a global function for a minor optimization (inlining, avoiding a closure).
func noopPrint(*starlark.Thread, string) {}

// run executes function with a given name with given arguments and returns result and fatal error.
// ThreadName is used only for debugging.
// Print is a user-suplied function for Starlark 'print'.
func (env *Env) run(funcName string, args starlark.Tuple, threadName string, printFunc PrintFunc) (starlark.Value, error) { //nolint:ireturn
	thread := &starlark.Thread{
		Name:  threadName,
		Print: noopPrint,
	}
	if printFunc != nil {
		thread.Print = func(t *starlark.Thread, msg string) {
			// make it look similar to starlark.CallStack.String
			fr := t.CallFrame(1)
			printFunc("thread "+t.Name+":", fr.Pos.String()+":", "in", fr.Name+":", msg)
		}
	}

	globals, err := env.p.Init(thread, env.predeclared)
	if err != nil {
		var eErr *starlark.EvalError
		if ok := errors.As(err, &eErr); ok {
			// tweak message, but keep original type, callstack, and cause
			eErr.Msg = fmt.Sprintf("thread %s: failed to init script: %s\n%s", threadName, eErr.Msg, eErr.CallStack)
			return nil, eErr
		}

		return nil, err
	}

	globals.Freeze()

	fn := globals[funcName]
	if fn == nil {
		return nil, fmt.Errorf("thread %s: function %s is not defined", threadName, funcName)
	}

	v, err := starlark.Call(thread, fn, args, nil)
	if err != nil {
		var eErr *starlark.EvalError
		if ok := errors.As(err, &eErr); ok {
			// tweak message, but keep original type, callstack, and cause
			eErr.Msg = fmt.Sprintf("thread %s: failed to execute function %s: %s\n%s", threadName, funcName, eErr.Msg, eErr.CallStack)
			return nil, eErr
		}

		return nil, err
	}

	v.Freeze()

	return v, nil
}

// Run executes function 'check_context' with given query results and additional funcs known as 'context'.
// Id is used to separate that execution from other and used only for debugging.
// Print is a user-suplied Starlark 'print' function implementation.
func (env *Env) Run(id string, input any, contextFuncs map[string]GoFunc, printFunc PrintFunc) ([]check.Result, error) {
	var err error

	rows, err := prepareInput(input)
	if err != nil {
		return nil, err
	}

	context := starlark.NewDict(len(contextFuncs))
	for n, f := range contextFuncs {
		err = context.SetKey(starlark.String(n), starlark.NewBuiltin(n, makeFunc(f)))
		if err != nil {
			return nil, err
		}
	}

	context.Freeze()

	var output starlark.Value

	output, err = env.run("check_context", starlark.Tuple{rows, context}, id, printFunc)
	if err != nil {
		// thread id is already present
		return nil, err
	}

	res, err := parseOutput(output)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// prepareInput converts go types to starlark types.
func prepareInput(input any) (starlark.Value, error) { //nolint:ireturn
	l, err := goToStarlark(input)
	if err != nil {
		return nil, err
	}

	l.Freeze()

	return l, nil
}

// parseScriptOutput parses and validates script output and returns slice of Results.
func parseOutput(v starlark.Value) ([]check.Result, error) {
	gv, err := starlarkToGo(v)
	if err != nil {
		return nil, err
	}

	switch gv := gv.(type) {
	case []any:
		res := make([]check.Result, len(gv))
		for i, el := range gv {
			m, ok := el.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("failed to parse script output: result %d has wrong type: %T", i, el)
			}

			r, err := convertResult(m)
			if err != nil {
				return nil, err
			}

			res[i] = *r
		}

		return res, nil

	case string:
		return nil, fmt.Errorf("script returned error: %s", gv)

	default:
		return nil, fmt.Errorf("failed to parse script output: %[1]v (%[1]T)", gv)
	}
}

// getField returns m[key] if it is present and a string, empty string if absent, or error otherwise.
func getField(m map[string]any, key string) (string, error) {
	v, ok := m[key]
	if !ok {
		return "", nil
	}

	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("%[1]q has wrong type: %[2]T (%[2]v)", key, v)
	}

	return s, nil
}

func convertResult(m map[string]any) (*check.Result, error) {
	summary, err := getField(m, "summary")
	if err != nil {
		return nil, err
	}

	description, err := getField(m, "description")
	if err != nil {
		return nil, err
	}

	readMoreURL, err := getField(m, "read_more_url")
	if err != nil {
		return nil, err
	}

	severity, err := getField(m, "severity")
	if err != nil {
		return nil, err
	}

	var labels map[string]string

	l, ok := m["labels"]
	if ok {
		lm, ok := l.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("labels field has wrong type: %[1]T (%[1]v)", l)
		}

		labels = make(map[string]string, len(lm))
		for lk := range lm {
			lv, err := getField(lm, lk)
			if err != nil {
				return nil, err
			}

			labels[lk] = lv
		}
	}

	res := &check.Result{
		Summary:     summary,
		Description: description,
		ReadMoreURL: readMoreURL,
		Severity:    common.ParseSeverity(severity),
		Labels:      labels,
	}

	err = res.Validate()
	if err != nil {
		return nil, err
	}

	return res, nil
}

// CheckGlobals checks for the presence of `check` and `check_context` functions.
func CheckGlobals(c *check.Check, predeclaredFuncs map[string]GoFunc) error {
	predeclared := make(starlark.StringDict, len(predeclaredFuncs))

	for n, f := range predeclaredFuncs {
		predeclared[n] = starlark.NewBuiltin(n, makeFunc(f))
	}

	predeclared.Freeze()

	var thread starlark.Thread

	globals, err := starlark.ExecFile(&thread, "", c.Script, predeclared)
	if err != nil {
		return err
	}

	if c.Version == 1 {
		if _, ok := globals["check"].(*starlark.Function); !ok {
			return fmt.Errorf("%s: no `check` function found", c.Name)
		}
	}

	if _, ok := globals["check_context"].(*starlark.Function); !ok {
		return fmt.Errorf("%s: no `check_context` function found", c.Name)
	}

	return nil
}
