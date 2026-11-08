//go:build !full

package auth

import "github.com/gin-gonic/gin"

// setupAccounts reads accs.db once, at start. The full profile re-reads it when
// it changes (accounts_full.go).
func setupAccounts(engine *gin.Engine) {
	accs := getAccounts()
	if accs == nil {
		return
	}
	engine.Use(BasicAuth(accs))
}
