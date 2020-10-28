package linuxbox

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/alessio/shellescape"
	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/communicator/ssh"
)

var (
	errPathNotExist = errors.New("Path doesn't exist") // asuming read permission is allowed
	errNil          = errors.New("unexpected nil object")
)

type linuxBox struct {
	communicator *ssh.Communicator
	connInfo     map[string]string
}

func (p linuxBox) exec(cmd *remote.Cmd) (err error) {
	if err = p.communicator.Start(cmd); err != nil {
		return
	}
	return cmd.Wait()
}

type permission struct {
	owner uint16
	group uint16
	mode  string
}

func (l linuxBox) setPermission(ctx context.Context, path string, p permission) (err error) {
	pathSafe := shellescape.Quote(path)
	cmd := fmt.Sprintf(`sh -c "chown %d:%d %s && chmod %s %s"`,
		p.owner, p.group, pathSafe, p.mode, pathSafe)
	return l.exec(&remote.Cmd{Command: cmd})
}

func (l linuxBox) getPermission(ctx context.Context, path string) (p permission, err error) {
	stdout := new(bytes.Buffer)
	cmd := remote.Cmd{
		Command: fmt.Sprintf(`stat -c '%%u %%g %%a' %s`, shellescape.Quote(path)),
		Stdout:  stdout,
	}
	err = l.exec(&cmd)
	var exitError *remote.ExitError
	if errors.As(err, &exitError) {
		err = errPathNotExist
		return
	}
	if err != nil {
		return
	}

	out, err := stdout.ReadString('\n')
	if err != nil {
		return
	}
	parts := strings.Split(strings.TrimSpace(out), " ")
	if len(parts) != 3 {
		err = fmt.Errorf("malformed output of %q: %q", cmd.Command, out)
		return
	}
	owner, err := strconv.ParseUint(parts[0], 10, 16)
	if err != nil {
		err = fmt.Errorf("while parsing owner id %q: %w", parts[0], err)
		return
	}
	group, err := strconv.ParseUint(parts[1], 10, 16)
	if err != nil {
		err = fmt.Errorf("while parsing group id %q: %w", parts[0], err)
		return
	}
	p = permission{
		owner: uint16(owner),
		group: uint16(group),
		mode:  parts[2],
	}
	return
}

func (l linuxBox) reservePath(ctx context.Context, path string) (err error) {
	var exitError *remote.ExitError
	cmd := fmt.Sprintf("[ ! -e %s ]", shellescape.Quote(path))
	if err = l.exec(&remote.Cmd{Command: cmd}); errors.As(err, &exitError) {
		return fmt.Errorf("path '%s' exist", path)
	}
	return
}

func (l linuxBox) mkdirp(ctx context.Context, path string) (err error) {
	cmd := fmt.Sprintf(`mkdir -p %s`, shellescape.Quote(path))
	return l.exec(&remote.Cmd{Command: cmd})
}

func (l linuxBox) cat(ctx context.Context, path string) (s string, err error) {
	stdout := new(bytes.Buffer)
	cmd := fmt.Sprintf("cat %s", shellescape.Quote(path))
	if err = l.exec(&remote.Cmd{Command: cmd, Stdout: stdout}); err != nil {
		return
	}
	return stdout.String(), nil
}

func (l linuxBox) mv(ctx context.Context, old, new string) (err error) {
	cmd := fmt.Sprintf(`mv %s %s`, shellescape.Quote(old), shellescape.Quote(new))
	return l.exec(&remote.Cmd{Command: cmd})
}

func (l linuxBox) remove(ctx context.Context, path, recyclePath string) (err error) {
	if path == "" {
		return
	}

	var cmd string
	path = shellescape.Quote(path)
	if recyclePath != "" {
		recycleFolder := shellescape.Quote(fmt.Sprintf("%s/%d", recyclePath, time.Now().Unix()))
		cmd = fmt.Sprintf(`sh -c "[ ! -e %s ] || { mkdir -p %s && mv %s %s; }"`, path, recycleFolder, path, recycleFolder)
	} else {
		cmd = fmt.Sprintf(`sh -c "[ ! -e %s ] || rm -rf %s"`, path, path)
	}
	return l.exec(&remote.Cmd{Command: cmd})
}
