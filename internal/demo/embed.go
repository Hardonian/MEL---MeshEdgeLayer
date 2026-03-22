package demo

import _ "embed"

//go:embed default_sandbox.json
var defaultSandboxJSON []byte

// DefaultSandboxConfigBytes returns the embedded demo sandbox config JSON.
func DefaultSandboxConfigBytes() []byte {
	return defaultSandboxJSON
}
