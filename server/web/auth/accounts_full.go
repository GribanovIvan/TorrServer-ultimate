//go:build full

package auth

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"

	"server/log"
	"server/settings"
)

// accs.db is re-read when it changes, so accounts can be added or removed
// without a restart. Idea from reyohoho/TorrServer (1496fa0a).

// accsPollInterval is how often accs.db is checked for changes. Tests shorten it.
var accsPollInterval = 5 * time.Second

var (
	accsPairs     atomic.Pointer[authPairs]
	accsFile      atomic.Pointer[string]
	accsWatchOnce sync.Once
)

func setupAccounts(engine *gin.Engine) {
	path := filepath.Join(settings.Path, "accs.db")
	accsFile.Store(&path)
	loadAccounts(path)
	engine.Use(func(c *gin.Context) {
		// Without accs.db there is nobody to log in as: CheckAuth refuses, as
		// in the other profiles.
		if p := accsPairs.Load(); p != nil {
			pairsAuth(*p)(c)
		}
	})
	interval := accsPollInterval
	accsWatchOnce.Do(func() { go watchAccounts(interval) })
}

func loadAccounts(path string) {
	accs := readAccounts(path)
	if accs == nil {
		accsPairs.Store(nil)
		return
	}
	pairs := processAccounts(accs)
	accsPairs.Store(&pairs)
}

// watchAccounts reloads accs.db after its modification time or size changes.
func watchAccounts(interval time.Duration) {
	var lastMod time.Time
	var lastSize int64 = -1
	for {
		time.Sleep(interval)
		path := *accsFile.Load()
		fi, err := os.Stat(path)
		if err != nil {
			if lastSize != -1 {
				log.TLogln("accs.db is gone, no accounts until it is back")
				accsPairs.Store(nil)
				lastMod, lastSize = time.Time{}, -1
			}
			continue
		}
		if fi.ModTime().Equal(lastMod) && fi.Size() == lastSize {
			continue
		}
		first := lastSize == -1
		lastMod, lastSize = fi.ModTime(), fi.Size()
		// A file caught half-written does not parse: keep the accounts we have.
		accs := readAccounts(path)
		if accs == nil {
			log.TLogln("accs.db unreadable, keeping the previous accounts")
			continue
		}
		pairs := processAccounts(accs)
		accsPairs.Store(&pairs)
		if !first {
			log.TLogln("accs.db changed, accounts reloaded")
		}
	}
}
