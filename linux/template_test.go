package linux

import (
	"fmt"
	"text/template"

	"github.com/MakeNowJust/heredoc"
	"github.com/Masterminds/sprig"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"go.uber.org/zap/buffer"
)

func tCompileTemplate(tmpl string, data interface{}) (s string, err error) {
	t, err := template.New("tf").Funcs(sprig.TxtFuncMap()).Parse(tmpl)
	if err != nil {
		return
	}
	buff := new(buffer.Buffer)
	err = t.Execute(buff, data)
	if err != nil {
		return
	}
	return buff.String(), nil
}

type tfmap map[string]string

func (m tfmap) With(k, v string) tfmap {
	m[k] = v
	return m
}

func (m tfmap) Without(keys ...string) tfmap {
	for _, k := range keys {
		delete(m, k)
	}
	return m
}
func (m tfmap) Copy() tfmap {
	c := tfmap{}
	for k, v := range m {
		c[k] = v
	}
	return c
}

func (m tfmap) Serialize() (s string, err error) {
	tf := heredoc.Doc(`
		{{ range $key, $value := . }}
		    {{- if $value | contains "\n" }}
		        {{- $key | nindent 0 }} = <<-EOF
		            {{- $value | nindent 4 }}
		        {{- "EOF" | nindent 0 }}
		    {{- else }}
		        {{- $key | nindent 0 }} = {{ $value }}
		    {{- end}}
		{{- end}}
	`)

	return tCompileTemplate(tf, m)
}

func tNewTFMapDirectory() tfmap {
	m := tfmap{}
	m[attrDirectoryPath] = fmt.Sprintf(`"/tmp/linux/%s"`, acctest.RandString(16))
	m[attrDirectoryOwner] = fmt.Sprintf("%d", acctest.RandInt()%1000+1000)
	m[attrDirectoryGroup] = fmt.Sprintf("%d", acctest.RandInt()%1000+1000)
	m[attrDirectoryMode] = `"755"`
	return m
}

func tNewTFMapFile() tfmap {
	m := tfmap{}
	m[attrFilePath] = fmt.Sprintf(`"/tmp/linux/%s"`, acctest.RandString(16))
	for i := 0; i < acctest.RandInt()%10+2; i++ {
		m[attrFileContent] = m[attrFileContent] + acctest.RandString(10+i) + " " + acctest.RandString(10+i) + "\n"
	}
	m[attrFileOwner] = fmt.Sprintf("%d", acctest.RandInt()%1000+1000)
	m[attrFileGroup] = fmt.Sprintf("%d", acctest.RandInt()%1000+1000)
	m[attrFileMode] = `"755"`
	return m
}

type tfList []string

func (l tfList) Copy() (c tfList) {
	return append(c, l...)
}

func (l tfList) Serialize() (s string, err error) {
	tf := heredoc.Doc(`
		{{ range  $value := . }}
		    {{- if $value | contains "\n" }}
		        {{- "<<-EOF" | nindent 0 }}
		            {{- $value | nindent 4 }}
		        {{- "EOF" | nindent 0 }},
		    {{- else }}
		        {{ $value }},
		    {{- end}}
		{{- end}}
	`)

	return tCompileTemplate(tf, l)
}

type tfScript struct {
	Triggers             tfmap
	Environment          tfmap
	SensitiveEnvironment tfmap
	WorkingDirectory     string
	Interpreter          tfList
	LifecycleCommands    tfmap
}

func (t tfScript) Copy(modifers ...func(*tfScript)) (c tfScript) {
	c.Triggers = t.Triggers.Copy()
	c.Environment = t.Environment.Copy()
	c.SensitiveEnvironment = t.SensitiveEnvironment.Copy()
	c.WorkingDirectory = t.WorkingDirectory
	c.Interpreter = t.Interpreter.Copy()
	c.LifecycleCommands = t.LifecycleCommands.Copy()

	for _, m := range modifers {
		m(&c)
	}
	return
}

func (t tfScript) Serialize() (s string, err error) {
	data := struct {
		Key   map[string]string
		Value tfScript
	}{
		Key: map[string]string{
			"Triggers":             attrScriptTriggers,
			"Environment":          attrScriptEnvironment,
			"SensitiveEnvironment": attrScriptSensitiveEnvironment,
			"WorkingDirectory":     attrScriptWorkingDirectory,
			"Interpreter":          attrScriptInterpreter,
			"LifecycleCommands":    attrScriptLifecycleCommands,
		},
		Value: t,
	}
	tf := heredoc.Doc(`
		{{ if .Value.LifecycleCommands -}}
		    {{- .Key.LifecycleCommands | nindent 0 }} {
		        {{- .Value.LifecycleCommands.Serialize | nindent 4 }}
		    {{- "}" | nindent 0 -}}
		{{ end -}}

		{{ if .Value.Triggers -}}
		    {{- .Key.Triggers | nindent 0 }} = {
		        {{- .Value.Triggers.Serialize | nindent 4 }}
		    {{- "}" | nindent 0 -}}
		{{ end -}}

		{{ if .Value.Environment -}}
		    {{- .Key.Environment | nindent 0 }} = {
		        {{- .Value.Environment.Serialize | nindent 4 }}
		    {{- "}" | nindent 0 -}}
		{{ end -}}

		{{ if .Value.SensitiveEnvironment -}}
		    {{- .Key.SensitiveEnvironment | nindent 0 }} = {
		        {{- .Value.SensitiveEnvironment.Serialize | nindent 4 }}
		    {{- "}" | nindent 0 -}}
		{{ end -}}

		{{ if .Value.WorkingDirectory }}
		    {{- .Key.WorkingDirectory | nindent 0 }} = {{ .Value.WorkingDirectory }}
		{{ end }}

		{{ if .Value.Interpreter -}}
		    {{- .Key.Interpreter }} = [
		        {{- .Value.Interpreter.Serialize | nindent 4 }}
		    {{- "]" | nindent 0 -}}
		{{- end -}}
	`)
	return tCompileTemplate(tf, data)
}

type tfConf struct {
	Provider  tfmap
	File      tfmap
	Directory tfmap
	Script    tfScript
	Extra     tfmap
}

func (c tfConf) compile(tmpl string) (string, error) {
	return tCompileTemplate(tmpl, c)
}

func (c tfConf) Copy(modifers ...func(*tfConf)) (n tfConf) {
	n.Provider = c.Provider.Copy()
	n.File = c.File.Copy()
	n.Directory = c.Directory.Copy()
	n.Script = c.Script.Copy()
	n.Extra = c.Extra.Copy()
	for _, m := range modifers {
		m(&n)
	}
	return
}
