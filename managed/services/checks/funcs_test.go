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

package checks

import (
	"strings"
	"testing"

	"github.com/percona/saas/pkg/check"
	"github.com/percona/saas/pkg/common"
	"github.com/percona/saas/pkg/starlark"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersion(t *testing.T) {
	script := strings.TrimSpace(`
def check_context(rows, context):
    v = parse_version(rows[0].get("version"))
    print("v =", v)

    s = format_version_num(v["num"])
    print("s =", s)

    return [{
        "summary": s,
        "severity": "warning",
        "labels": {
            "major": str(v["major"]),
            "minor": str(v["minor"]),
            "patch": str(v["patch"]),
            "rest":  str(v["rest"]),
            "num":   str(v["num"]),
        }
    }]
	`)
	funcs, err := GetFuncsForVersion(1)
	require.NoError(t, err)
	env, err := starlark.NewEnv(t.Name(), script, funcs)
	require.NoError(t, err)

	input := []map[string]interface{}{
		{"version": int64(1)},
	}
	res, err := env.Run("type", input, nil, t.Log)
	expectedErr := strings.TrimSpace(`
thread type: failed to execute function check_context: parse_version: expected string argument, got int64 (1)
Traceback (most recent call last):
  TestVersion:2:22: in check_context
  <builtin>: in parse_version
	`) + "\n"
	assert.EqualError(t, err, expectedErr)
	assert.Empty(t, res)

	input = []map[string]interface{}{
		{"version": "foo"},
	}
	res, err = env.Run("foo", input, nil, t.Log)
	expectedErr = strings.TrimSpace(`
thread foo: failed to execute function check_context: parse_version: failed to parse "foo"
Traceback (most recent call last):
  TestVersion:2:22: in check_context
  <builtin>: in parse_version
	`) + "\n"
	assert.EqualError(t, err, expectedErr)
	assert.Empty(t, res)

	input = []map[string]interface{}{
		{"version": "5.7.20-19-log"},
	}
	res, err = env.Run("valid", input, nil, t.Log)
	require.NoError(t, err)
	expected := []check.Result{{
		Summary:  "5.7.20",
		Severity: common.Warning,
		Labels: map[string]string{
			"major": "5",
			"minor": "7",
			"patch": "20",
			"rest":  "-19-log",
			"num":   "50720",
		},
	}}
	assert.Equal(t, expected, res)
}

func TestAdditionalContext(t *testing.T) {
	t.Parallel()
	predeclaredFuncs, err := GetFuncsForVersion(1)
	require.NoError(t, err)
	contextFuncs := GetAdditionalContext()

	testCases := []struct {
		name   string
		script string
		err    string
		result []check.Result
	}{
		{
			name: "too many args",
			script: strings.TrimSpace(`
def check_context(rows, context):
    ip_is_private = context.get("ip_is_private", fail)

    return [{
        "summary": "IP Address Check",
		"severity": "warning",
		"description": "is_private: {}".format(ip_is_private(1, 2))
    }]
	`),
			err: strings.TrimSpace(`
thread too many args: failed to execute function check_context: ip_is_private: expected 1 argument, got 2
Traceback (most recent call last):
  TestAdditionalContext/too_many_args:7:55: in check_context
  <builtin>: in ip_is_private
	`) + "\n",
			result: nil,
		},
		{
			name: "invalid arg",
			script: strings.TrimSpace(`
def check_context(rows, context):
    ip_is_private = context.get("ip_is_private", fail)

    return [{
        "summary": "IP Address Check",
		"severity": "warning",
		"description": "is_private: {}".format(ip_is_private("some-address"))
    }]
	`),
			err: "",
			result: []check.Result{{
				Summary:     "IP Address Check",
				Severity:    common.Warning,
				Description: "is_private: None",
			}},
		},
		{
			name: "invalid arg type",
			script: strings.TrimSpace(`
def check_context(rows, context):
    ip_is_private = context.get("ip_is_private", fail)

    return [{
        "summary": "IP Address Check",
		"severity": "warning",
		"description": "is_private: {}".format(ip_is_private(1))
    }]
	`),
			err: strings.TrimSpace(`
thread invalid arg type: failed to execute function check_context: ip_is_private: expected string argument, got int64 (1)
Traceback (most recent call last):
  TestAdditionalContext/invalid_arg_type:7:55: in check_context
  <builtin>: in ip_is_private
		`) + "\n",
			result: nil,
		},
		{
			name: "valid argument",
			script: strings.TrimSpace(`
def check_context(rows, context):
    ip_is_private = context.get("ip_is_private", fail)

    return [{
        "summary": "IP Address Check",
		"severity": "warning",
		"description": "is_private: {}".format(ip_is_private("127.0.0.1"))
    }]
	`),
			err: "",
			result: []check.Result{{
				Summary:     "IP Address Check",
				Severity:    common.Warning,
				Description: "is_private: True",
			}},
		},
		{
			name: "valid ipv6 argument",
			script: strings.TrimSpace(`
def check_context(rows, context):
    ip_is_private = context.get("ip_is_private", fail)

    return [{
        "summary": "IP Address Check",
		"severity": "warning",
		"description": "is_private: {}".format(ip_is_private("0:0:0:0:0:0:0:1"))
    }]
	`),
			err: "",
			result: []check.Result{{
				Summary:     "IP Address Check",
				Severity:    common.Warning,
				Description: "is_private: True",
			}},
		},
		{
			name: "public ip address",
			script: strings.TrimSpace(`
def check_context(rows, context):
    ip_is_private = context.get("ip_is_private", fail)

    return [{
        "summary": "IP Address Check",
		"severity": "warning",
		"description": "is_private: {}".format(ip_is_private("1.1.1.1")),
    }]
	`),
			err: "",
			result: []check.Result{{
				Summary:     "IP Address Check",
				Severity:    common.Warning,
				Description: "is_private: False",
			}},
		},
		{
			name: "private network",
			script: strings.TrimSpace(`
def check_context(rows, context):
    ip_is_private = context.get("ip_is_private", fail)

    return [{
        "summary": "IP Address Check",
		"severity": "warning",
		"description": "is_private: {}".format(ip_is_private("10.0.0.0/9"))
    }]
	`),
			err: "",
			result: []check.Result{{
				Summary:     "IP Address Check",
				Severity:    common.Warning,
				Description: "is_private: True",
			}},
		},
		{
			name: "public network",
			script: strings.TrimSpace(`
def check_context(rows, context):
    ip_is_private = context.get("ip_is_private", fail)

    return [{
        "summary": "IP Address Check",
		"severity": "warning",
		"description": "is_private: {}".format(ip_is_private("192.88.99.0/24"))
    }]
	`),
			err: "",
			result: []check.Result{{
				Summary:     "IP Address Check",
				Severity:    common.Warning,
				Description: "is_private: False",
			}},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			env, err := starlark.NewEnv(t.Name(), tc.script, predeclaredFuncs)
			require.NoError(t, err)
			res, err := env.Run(tc.name, nil, contextFuncs, t.Log)
			if res != nil {
				require.NoError(t, err)
				assert.Equal(t, tc.result, res)
			} else {
				assert.EqualError(t, err, tc.err)
				assert.Empty(t, res)
			}
		})
	}
}
