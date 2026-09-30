package service

import "sync"

// callLocks — мьютекс на звонок. Команды ринг-фазы, вебхуки LiveKit, отложенная
// сверка состава и истечение дозвона приходят параллельно, а каждая читает
// снимок звонка и пишет его обратно: без сериализации «принять» записывало
// active поверх уже выставленного missed. Записи удаляются, когда звонок
// больше никто не держит.
type callLocks struct {
	mu sync.Mutex
	m  map[int64]*callLock
}

type callLock struct {
	mu   sync.Mutex
	refs int
}

func newCallLocks() *callLocks { return &callLocks{m: make(map[int64]*callLock)} }

// lock — захватить звонок; вернуть функцию освобождения.
func (l *callLocks) lock(callID int64) func() {
	l.mu.Lock()
	cl, ok := l.m[callID]
	if !ok {
		cl = &callLock{}
		l.m[callID] = cl
	}
	cl.refs++
	l.mu.Unlock()

	cl.mu.Lock()
	return func() {
		cl.mu.Unlock()
		l.mu.Lock()
		cl.refs--
		if cl.refs == 0 {
			delete(l.m, callID)
		}
		l.mu.Unlock()
	}
}
