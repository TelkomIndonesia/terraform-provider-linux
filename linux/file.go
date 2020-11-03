package linux

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/alessio/shellescape"
	"github.com/hashicorp/terraform/communicator/remote"
	"golang.org/x/net/context"
)

type file struct {
	path       string
	content    string
	permission permission

	ignoreContent bool
	overwrite     bool
	recyclePath   string
}

func (l *linux) readFile(ctx context.Context, path string, ignoreContent bool) (f *file, err error) {
	perm, err := l.getPermission(ctx, path)
	if err != nil {
		return
	}

	f = &file{path: path, permission: perm, ignoreContent: ignoreContent}
	if f.ignoreContent {
		return
	}

	f.content, err = l.cat(ctx, f.path)
	return
}

func (l *linux) createFile(ctx context.Context, f *file) (err error) {
	if f == nil {
		return errNil
	}

	if !f.overwrite {
		if err = l.reservePath(ctx, f.path); err != nil {
			return
		}
	}

	err = l.mkdirp(ctx, filepath.Dir(f.path))
	if err != nil {
		return
	}

	switch f.ignoreContent {
	case false:
		err = l.upload(ctx, f.path, strings.NewReader(f.content))
	case true:
		pathSafe := shellescape.Quote(f.path)
		err = l.exec(ctx, &remote.Cmd{Command: fmt.Sprintf(`sh -c "touch %s && [ -f %s ]"`, pathSafe, pathSafe)})
	}
	if err != nil {
		return
	}

	return l.setPermission(ctx, f.path, f.permission)
}

func (l *linux) deleteFile(ctx context.Context, f *file) (err error) {
	if f == nil {
		return
	}
	return l.remove(ctx, f.path, f.recyclePath)
}

func (l *linux) updateFile(ctx context.Context, old, new *file) (err error) {
	if old == nil {
		return l.createFile(ctx, new)
	}
	if new == nil {
		return l.deleteFile(ctx, old)
	}

	if old.path != new.path {
		if !new.overwrite {
			if err = l.reservePath(ctx, new.path); err != nil {
				return
			}
		}
		err = l.mv(ctx, old.path, new.path)
		if err != nil {
			return
		}
	}

	f := &file{}
	*f = *new
	f.overwrite = true
	return l.createFile(ctx, f)
}
