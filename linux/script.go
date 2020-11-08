package linux

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

func (e env) serialize(sep string) string {
	b := strings.Builder{}
	first := true
	for k, v := range e {
		if first {
			first = false
		} else {
			b.WriteString(sep)
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
	l *linux

	workdir     string
	env         env
	interpreter []string
	body        string

	stdin io.Reader
}

func (sc *script) exec(ctx context.Context) (res string, err error) {
	path := shellescape.Quote(sc.l.scriptPath(ctx))
	err = sc.l.uploadScript(ctx, path, strings.NewReader(sc.body))
	if err != nil {
		return
	}
	defer func() { _ = sc.l.remove(ctx, path, "") }()

	cmd := fmt.Sprintf(`{ %s && %s && %s %s ;}`,
		shellescape.QuoteCommand([]string{"mkdir", "-p", sc.workdir}),
		shellescape.QuoteCommand([]string{"cd", sc.workdir}),
		sc.env.inline(), shellescape.QuoteCommand(append(sc.interpreter, path)),
	)
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	err = sc.l.exec(ctx,
		&remote.Cmd{
			Command: cmd,
			Stdin:   sc.stdin,
			Stdout:  stdout,
			Stderr:  stderr,
		},
	)
	if err != nil {
		err = fmt.Errorf("stderr: %s; error: %w", stderr, err)
		return
	}
	return stdout.String(), nil
}
