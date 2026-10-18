package update

import (
	"os"
	"saveme/internal/config"
	"testing"
)

func testIdentity(t *testing.T) config.Config {
	t.Helper()
	uid := uint32(os.Getuid())
	return config.Config{AllowedUID: &uid}
}
func testEndpoint(path string) string { return path }
