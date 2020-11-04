package linux

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alessio/shellescape"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/communicator/ssh"
	"github.com/hashicorp/terraform/terraform"
)

var (
	errPathNotExist = errors.New("Path doesn't exist") // asuming read permission is allowed
	errNil          = errors.New("unexpected nil object")
)

type linux struct {
	connInfo map[string]string

	comm     *ssh.Communicator
	commErr  error
	commOnce sync.Once
}

func (l *linux) init(ctx context.Context) error {
	l.comm, l.commErr = ssh.NewNoPty(&terraform.InstanceState{Ephemeral: terraform.EphemeralState{
		ConnInfo: l.connInfo,
	}})
	if l.commErr != nil {
		return l.commErr
	}

	l.commErr = l.comm.Connect(nil)
	return l.commErr
}

func (l *linux) communicator(ctx context.Context) (*ssh.Communicator, error) {
	l.commOnce.Do(func() {
		err := resource.RetryContext(ctx, 5*time.Minute, func() *resource.RetryError {
			var errNet net.Error
			switch err := l.init(ctx); {
			default:
				return nil

			case errors.As(err, &errNet):
				return resource.RetryableError(errNet)

			case err != nil:
				return resource.NonRetryableError(err)
			}
		})

		if l.commErr == nil {
			l.commErr = err
		}
	})

	return l.comm, l.commErr
}

func (l *linux) exec(ctx context.Context, cmd *remote.Cmd) (err error) {
	c, err := l.communicator(ctx)
	if err != nil {
		return
	}
	if err = c.Start(cmd); err != nil {
		return
	}
	return cmd.Wait()
}

func (l *linux) upload(ctx context.Context, path string, input io.Reader) (err error) {
	c, err := l.communicator(ctx)
	if err != nil {
		return
	}
	return c.Upload(path, input)
}

func (l *linux) uploadScript(ctx context.Context, path string, input io.Reader) (err error) {
	c, err := l.communicator(ctx)
	if err != nil {
		return
	}
	return c.UploadScript(path, input)
}

func (l *linux) scriptPath(ctx context.Context) string {
	c, err := l.communicator(ctx)
	if err != nil {
		return ""
	}
	return c.ScriptPath()
}

type permission struct {
	owner uint16
	group uint16
	mode  string
}

func (l *linux) setPermission(ctx context.Context, path string, p permission) (err error) {
	pathSafe := shellescape.Quote(path)
	cmd := fmt.Sprintf(`sh -c "chown %d:%d %s && chmod %s %s"`,
		p.owner, p.group, pathSafe, p.mode, pathSafe)
	return l.exec(ctx, &remote.Cmd{Command: cmd})
}

func (l *linux) getPermission(ctx context.Context, path string) (p permission, err error) {
	stdout := new(bytes.Buffer)
	cmd := remote.Cmd{
		Command: fmt.Sprintf(`stat -c '%%u %%g %%a' %s`, shellescape.Quote(path)),
		Stdout:  stdout,
	}
	err = l.exec(ctx, &cmd)
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

func (l *linux) reservePath(ctx context.Context, path string) (err error) {
	var exitError *remote.ExitError
	cmd := fmt.Sprintf("[ ! -e %s ]", shellescape.Quote(path))
	if err = l.exec(ctx, &remote.Cmd{Command: cmd}); errors.As(err, &exitError) {
		return fmt.Errorf("path '%s' exist", path)
	}
	return
}

func (l *linux) mkdirp(ctx context.Context, path string) (err error) {
	cmd := fmt.Sprintf(`mkdir -p %s`, shellescape.Quote(path))
	return l.exec(ctx, &remote.Cmd{Command: cmd})
}

func (l *linux) cat(ctx context.Context, path string) (s string, err error) {
	stdout := new(bytes.Buffer)
	cmd := fmt.Sprintf("cat %s", shellescape.Quote(path))
	if err = l.exec(ctx, &remote.Cmd{Command: cmd, Stdout: stdout}); err != nil {
		return
	}
	return stdout.String(), nil
}

func (l *linux) mv(ctx context.Context, old, new string) (err error) {
	cmd := fmt.Sprintf(`mv %s %s`, shellescape.Quote(old), shellescape.Quote(new))
	return l.exec(ctx, &remote.Cmd{Command: cmd})
}

func (l *linux) remove(ctx context.Context, path, recyclePath string) (err error) {
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
	return l.exec(ctx, &remote.Cmd{Command: cmd})
}
