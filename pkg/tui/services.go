package tui

import (
	"zai-api-client/pkg/accounts"
	"zai-api-client/pkg/client"
	"zai-api-client/pkg/coding"
)

// Config bundles the already-constructed service clients that every screen
// needs. Screens receive these via their constructor and never call
// getClient(), accounts.Load(), etc. themselves — that resolution happens
// once, in tui_cli.go, exactly like every other Cobra command.
type Config struct {
	Client   *client.Client
	Accounts *accounts.Store
	Coding   *coding.Store
}
