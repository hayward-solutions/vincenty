package integration

import (
	"os"
	"sync"
	"testing"

	"github.com/sitaware/api/internal/testutil"
)

var (
	env     *testutil.TestEnv
	envOnce sync.Once
	envErr  bool
)

// getEnv lazily initialises the shared test environment on first call.
// The environment persists until TestMain tears it down after all tests finish.
func getEnv(t *testing.T) *testutil.TestEnv {
	t.Helper()
	envOnce.Do(func() {
		// Setup uses t.Fatalf which will mark this test as failed, but
		// since we only call Fatalf (which calls runtime.Goexit for the
		// calling goroutine), env will remain nil. We detect that below.
		env = testutil.Setup(t)
	})
	if env == nil {
		t.Skip("test environment not available (setup failed)")
	}
	return env
}

func TestMain(m *testing.M) {
	code := m.Run()

	// Teardown after ALL tests have completed.
	if env != nil {
		env.Teardown()
	}

	os.Exit(code)
}
