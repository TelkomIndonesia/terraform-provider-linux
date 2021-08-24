package linux

import (
	"fmt"
	"sync"
)

type linuxPool struct {
	mut sync.Mutex

	def  *linux
	pool map[string]*linux
}

func (lp *linuxPool) getOrSet(id string, l *linux) (*linux, error) {
	lp.mut.Lock()
	defer lp.mut.Unlock()

	lg, ok := lp.pool[id]
	if !ok {
		lp.pool[id] = l
		return l, nil
	}
	if !lg.Equal(l) {
		return nil, fmt.Errorf("conflicting connection")
	}
	return lg, nil
}
