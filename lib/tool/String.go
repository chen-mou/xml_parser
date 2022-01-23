package tool

import (
	"strings"
	"text/template"
)

type String string

func (s String) Format(data map[string]interface{}) (out string, err error) {
	t := template.Must(template.New("").Parse(string(s)))
	builder := &strings.Builder{}
	if err = t.Execute(builder, data); err != nil {
		return
	}
	out = builder.String()
	return
}
