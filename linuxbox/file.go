package linuxbox

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/alessio/shellescape"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/spf13/cast"
	"golang.org/x/net/context"
)

var errPathNotExist = errors.New("Path doesn't exist") // asuming read permission is allowed
var errNil = errors.New("unexpected nil object")

type File struct {
	path          string
	content       string
	owner         uint16
	group         uint16
	mode          string
	ignoreContent bool
	replace       bool
	recyclePath   string
}

func newFileFromResourceData(rd *schema.ResourceData) (f *File) {
	if rd == nil {
		return
	}
	f = &File{
		path:          cast.ToString(rd.Get(attrFilePath)),
		content:       cast.ToString(rd.Get(attrFileContent)),
		owner:         cast.ToUint16(rd.Get(attrFileOwner)),
		group:         cast.ToUint16(rd.Get(attrFileGroup)),
		mode:          cast.ToString(rd.Get(attrFileMode)),
		ignoreContent: cast.ToBool(rd.Get(attrFileIgnoreContent)),
		replace:       cast.ToBool(rd.Get(attrFileReplace)),
		recyclePath:   cast.ToString(rd.Get(attrFileRecyclePath)),
	}
	return
}
func newDiffedFileFromResourceData(rd *schema.ResourceData) (old, new *File) {
	if rd == nil {
		return
	}
	old, new = &File{}, &File{}

	o, n := rd.GetChange(attrFilePath)
	old.path, new.path = cast.ToString(o), cast.ToString(n)

	o, n = rd.GetChange(attrFileContent)
	old.content, new.content = cast.ToString(o), cast.ToString(n)

	o, n = rd.GetChange(attrFileOwner)
	old.owner, new.owner = cast.ToUint16(o), cast.ToUint16(n)

	o, n = rd.GetChange(attrFileGroup)
	old.group, new.group = cast.ToUint16(o), cast.ToUint16(n)

	o, n = rd.GetChange(attrFileMode)
	old.mode, new.mode = cast.ToString(o), cast.ToString(n)

	o, n = rd.GetChange(attrFileIgnoreContent)
	old.ignoreContent, new.ignoreContent = cast.ToBool(o), cast.ToBool(n)

	o, n = rd.GetChange(attrFileReplace)
	old.replace, new.replace = cast.ToBool(o), cast.ToBool(n)

	o, n = rd.GetChange(attrFileRecyclePath)
	old.recyclePath, new.recyclePath = cast.ToString(o), cast.ToString(n)
	return
}

func (f *File) setResourceData(rd *schema.ResourceData) (err error) {
	if f == nil {
		rd.SetId("")
		return
	}

	if err = rd.Set(attrFilePath, f.path); err != nil {
		return
	}
	if err = rd.Set(attrFileContent, f.content); err != nil {
		return
	}
	if err = rd.Set(attrFileOwner, f.owner); err != nil {
		return
	}
	if err = rd.Set(attrFileGroup, f.group); err != nil {
		return
	}
	if err = rd.Set(attrFileMode, f.mode); err != nil {
		return
	}
	if err = rd.Set(attrFileIgnoreContent, f.ignoreContent); err != nil {
		return
	}
	if err = rd.Set(attrFileReplace, f.replace); err != nil {
		return
	}
	if err = rd.Set(attrFileRecyclePath, f.recyclePath); err != nil {
		return
	}
	return
}

func (p LinuxBox) ReadFile(ctx context.Context, path string) (f *File, err error) {
	stdout := new(bytes.Buffer)
	pathSafe := shellescape.Quote(path)

	{
		cmd := remote.Cmd{
			Command: fmt.Sprintf(`stat -c '%%u %%g %%a' %s`, pathSafe),
			Stdout:  stdout,
		}
		err = p.exec(&cmd)
		var exitError *remote.ExitError
		switch {
		case errors.As(err, &exitError):
			return nil, errPathNotExist

		case err != nil:
			return
		}

		out, err := stdout.ReadString('\n')
		if err != nil {
			return f, err
		}
		out = strings.TrimSpace(out)
		parts := strings.Split(out, " ")
		if len(parts) != 3 {
			return nil, fmt.Errorf("malformed output of %q: %q", cmd.Command, out)
		}
		owner, err := strconv.ParseUint(parts[0], 10, 16)
		if err != nil {
			return nil, fmt.Errorf("while parsing owner id %q: %w", parts[0], err)
		}
		group, err := strconv.ParseUint(parts[1], 10, 16)
		if err != nil {
			return nil, fmt.Errorf("while parsing group id %q: %w", parts[0], err)
		}
		f = &File{owner: uint16(owner), group: uint16(group), mode: parts[2], path: path}
	}

	{
		stdout.Reset()
		cmd := remote.Cmd{
			Command: fmt.Sprintf("cat %s | base64", pathSafe),
			Stdout:  stdout,
		}
		if err = p.exec(&cmd); err != nil {
			return
		}
		b, err := base64.StdEncoding.DecodeString(stdout.String())
		if err != nil {
			return f, err
		}
		f.content = string(b)
	}
	return
}

func (p LinuxBox) CreateFile(ctx context.Context, f *File) (err error) {
	if f == nil {
		return errNil
	}
	path := shellescape.Quote(f.path)

	if !f.replace {
		_, err := p.ReadFile(ctx, f.path)
		if err != nil && !errors.Is(err, errPathNotExist) {
			return fmt.Errorf("file at %s is already exist", f.path)
		}
	}

	if f.ignoreContent {
		cmd := fmt.Sprintf(`
			sh -c "
				mkdir -p $(dirname %s) &&
				touch %s &&
				cat %s > /dev/null &&
				chown %d:%d %s &&
				chmod %s %s
			"`,
			path,
			path,
			path,
			f.owner, f.group, path,
			f.mode, path)
		return p.exec(&remote.Cmd{Command: cmd})
	}

	err = p.exec(&remote.Cmd{Command: fmt.Sprintf("mkdir -p $(dirname %s)", path)})
	if err != nil {
		return
	}
	err = p.communicator.Upload(f.path, strings.NewReader(f.content))
	if err != nil {
		return
	}
	cmd := fmt.Sprintf(`sh -c "chown %d:%d %s && chmod %s %s"`,
		f.owner, f.group, path, f.mode, path)
	return p.exec(&remote.Cmd{Command: cmd})
}

func (p LinuxBox) UpdateFile(ctx context.Context, old, new *File) (err error) {
	if old == nil {
		return p.CreateFile(ctx, new)
	}
	if new == nil {
		return p.DeleteFile(ctx, old)
	}

	if old.path != new.path {
		opath, npath := shellescape.Quote(old.path), shellescape.Quote(new.path)
		cmd := fmt.Sprintf(`mv %s %s`, opath, npath)
		if err = p.exec(&remote.Cmd{Command: cmd}); err != nil {
			return
		}
	}

	f := &File{}
	*f = *new
	f.replace = true
	return p.CreateFile(ctx, f)
}

func (p LinuxBox) DeleteFile(ctx context.Context, f *File) (err error) {
	if f == nil {
		return
	}

	var cmd string
	path := shellescape.Quote(f.path)
	if f.recyclePath != "" {
		recycleFolder := shellescape.Quote(fmt.Sprintf("%s/%d", f.recyclePath, time.Now().Unix()))
		cmd = fmt.Sprintf(`sh -c "mkdir -p %s && mv %s %s"`, recycleFolder, path, recycleFolder)
	} else {
		cmd = fmt.Sprintf(`rm -f %s`, path)
	}
	return p.exec(&remote.Cmd{Command: cmd})
}
