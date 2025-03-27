package selector

import (
	"math/rand"
	"sync"
	"time"
)

var (
	random     = rand.New(rand.NewSource(time.Now().UnixNano()))
	randomLock sync.Mutex
)

// nextRandom 返回一个小于n的随机数
func nextRandom(n int64) int64 {
	if n <= 0 {
		return 0
	}
	randomLock.Lock()
	defer randomLock.Unlock()
	return random.Int63n(n)
}
