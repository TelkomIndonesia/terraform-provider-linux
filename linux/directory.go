package linux

import (
	"context"
	"fmt"

	"github.com/alessio/shellescape"
	"github.com/hashicorp/terraform/communicator/remote"
)

type directory struct {
	path       string
	permission permission

	overwrite   bool
	recyclePath string
}

func (l *linux) readDirectory(ctx context.Context, path string) (d *directory, err error) {
	perm, err := l.getPermission(ctx, path)
	if err != nil {
		return
	}
	d = &directory{path: path, permission: perm}
	return
}

func (l *linux) createDirectory(ctx context.Context, d *directory) (err error) {
	if d == nil {
		return errNil
	}

	if !d.overwrite {
		if err = l.reservePath(ctx, d.path); err != nil {
			return
		}
	}

	err = l.mkdirp(ctx, d.path)
	if err != nil {
		return
	}

	return l.setPermission(ctx, d.path, d.permission)
}

func (l *linux) deleteDirectory(ctx context.Context, f *directory) (err error) {
	if f == nil {
		return
	}
	return l.remove(ctx, f.path, f.recyclePath)
}

func (l *linux) updateDirectory(ctx context.Context, old, new *directory) (err error) {
	if old == nil {
		return l.createDirectory(ctx, new)
	}
	if new == nil {
		return l.deleteDirectory(ctx, old)
	}
	if old.path != new.path {
		if !new.overwrite {
			if err = l.reservePath(ctx, new.path); err != nil {
				return
			}
		}
		cmd := fmt.Sprintf(`sh -c '
				OLD_DIR=%s; NEW_DIR=%s;
				set -e

				mkdir -p ${NEW_DIR}
				if [ "$( ls -A ${OLD_DIR} )" ]; then
					mv ${OLD_DIR}/* ${NEW_DIR}/
				fi  
				rm -rf ${OLD_DIR}
			'`,
			shellescape.Quote(old.path), shellescape.Quote(new.path))
		err = l.exec(ctx, &remote.Cmd{Command: cmd})
		if err != nil {
			return
		}
	}

	d := &directory{}
	*d = *new
	d.overwrite = true
	return l.createDirectory(ctx, d)
}
