package linuxbox

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/alessio/shellescape"
	"github.com/hashicorp/terraform/communicator/remote"
)

type env map[string]string

func (e env) serialize(s string) string {
	b := strings.Builder{}
	first := true
	for k, v := range e {
		if first {
			first = false
		} else {
			b.WriteString(s)
		}

		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(shellescape.Quote(v))
	}
	return b.String()
}

func (e env) inline() string {
	return e.serialize(" ")
}

func (e env) multiline() string {
	return e.serialize("\n")
}

type script struct {
	l linuxBox

	workdir     string
	env         env
	interpreter []string
	body        string

	stdin io.Reader
}

func (sc *script) exec(ctx context.Context) (res string, err error) {
	path := shellescape.Quote(sc.l.communicator.ScriptPath())
	err = sc.l.communicator.UploadScript(path, strings.NewReader(sc.body))
	if err != nil {
		return
	}
	defer func() { _ = sc.l.remove(ctx, path, "") }()

	cmd := sc.env.inline() + " " + shellescape.QuoteCommand(sc.interpreter) + " " + path
	cmd = fmt.Sprintf(`sh -c 'cd %s && %s'`, shellescape.Quote(sc.workdir), cmd)
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	err = sc.l.exec(
		&remote.Cmd{
			Command: cmd,
			Stdin:   sc.stdin, //TODO: this doesn't seems to work
			Stdout:  stdout,
			Stderr:  stderr,
		},
	)
	if err != nil {
		err = fmt.Errorf("stderr: %s, error: %v", stderr, err)
	}
	return stdout.String(), nil
}
