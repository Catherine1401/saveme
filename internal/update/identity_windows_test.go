package update

import (
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"saveme/internal/config"
	"testing"
	"time"
)

func testIdentity(t *testing.T) config.Config {
	t.Helper()
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	return config.Config{AllowedSID: user.User.Sid.String()}
}
func testEndpoint(string) string {
	const format = `\\.\pipe\saveme-test-%d-%d`
	return fmt.Sprintf(format, os.Getpid(), time.Now().UnixNano())
}
