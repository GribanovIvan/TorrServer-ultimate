//go:build slim

package server

// The slim profile has neither HTTPS (a reverse proxy terminates TLS) nor the
// Telegram bot.

func prepareSSL() error { return nil }
func startTelegramBot() {}
