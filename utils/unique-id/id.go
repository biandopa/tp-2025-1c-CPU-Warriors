package uniqueid

import "sync"

type UniqueID struct {
	mu     sync.Mutex
	nextID int
}

func Init() *UniqueID {
	return &UniqueID{
		mu:     sync.Mutex{},
		nextID: 1, // El primer ID es 1
	}
}

func (u *UniqueID) GetUniqueID() int {
	u.mu.Lock()
	defer u.mu.Unlock()

	id := u.nextID
	u.nextID++
	return id
}
