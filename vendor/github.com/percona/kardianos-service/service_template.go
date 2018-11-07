package service

import (
	"strings"
)

var tf = map[string]interface{}{
	"cmd": func(s string) string {
		// Put command in single quotes, otherwise special characters like dollar ($) sign will be interpreted.
		return `'` + strings.Replace(s, `'`, `'"'"'`, -1) + `'`
	},
	"cmdSystemD": func(s string) string {
		s = strings.Replace(s, `%`, `%%`, -1)
		s = `"` + strings.Replace(s, `"`, `\"`, -1) + `"`
		return s
	},
	"cmdEscape": func(s string) string {
		return strings.Replace(s, " ", `\x20`, -1)
	},
	"envKey": func(env string) string {
		return strings.Split(env, "=")[0]
	},
	"envValue": func(env string) string {
		return strings.Join(strings.Split(env, "=")[1:], "=")
	},

	// http://supervisord.org/configuration.html?highlight=environment#file-format
	//   > This option can include the value %(here)s, which expands to the directory in which the supervisord
	//   > configuration file was found. Values containing non-alphanumeric characters should be quoted
	//   > (e.g. KEY="val:123",KEY2="val,456"). Otherwise, quoting the values is optional but recommended.
	//   > To escape percent characters, simply use two. (e.g. URI="/first%%20name")
	// See also https://github.com/Supervisor/supervisor/issues/328
	"envValueSupervisord": func(env string) string {
		v := strings.Join(strings.Split(env, "=")[1:], "=")
		return `'` + strings.Replace(v, "%", "%%", -1) + `'`
	},
}
