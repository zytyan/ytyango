//go:build frontenddev
// +build frontenddev

package g

// Minimal init for the frontenddev build tag. We only need logging and a
// mutable Config instance so that main_frontenddev can inject fixture
// parameters at runtime. No database is opened in this mode.
func init() {
	config = &Config{
		LogLevel: -1,
	}
	gWriteSyncer = initWriteSyncer()
}
