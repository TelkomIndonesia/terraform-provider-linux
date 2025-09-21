package linux

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"al.essio.dev/pkg/shellescape"
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

func (sc *script) scriptForUpload() (io.Reader, error) {
	if len(sc.interpreter) == 0 {
		return strings.NewReader(sc.body), nil
	}

	// prevent communicator from appending #!/bin/sh
	reader := bufio.NewReader(strings.NewReader(sc.body))
	prefix, err := reader.Peek(2)
	if err != nil {
		return nil, fmt.Errorf("Error reading script: %s", err)
	}
	var script bytes.Buffer
	if string(prefix) != "#!" {
		script.WriteString("#!\n")
	}
	_, _ = script.ReadFrom(reader)
	return &script, nil
}

func (sc *script) upload(ctx context.Context) (path string, err error) {
	path = shellescape.Quote(sc.l.scriptPath(ctx))
	s, err := sc.scriptForUpload()
	if err != nil {
		return "", err
	}
	if err := sc.l.uploadScript(ctx, path, s); err != nil {
		return "", err
	}
	return
}

func (sc *script) exec(ctx context.Context) (res string, err error) {
	path, err := sc.upload(ctx)
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
	err = sc.l.exec(ctx, &remote.Cmd{
		Command: cmd,
		Stdin:   sc.stdin,
		Stdout:  stdout,
		Stderr:  stderr,
	})
	if err != nil {
		err = fmt.Errorf("stderr: %s\nerror: %w", stderr, err)
		return
	}
	return stdout.String(), nil
}
