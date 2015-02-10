package testutil

import (
	"path/filepath"
	"time"

	"v.io/tools/lib/collect"
	"v.io/tools/lib/runutil"
	"v.io/tools/lib/util"
)

const (
	defaultPlaygroundTestTimeout = 5 * time.Minute
)

// vanadiumPlaygroundTest runs integration tests for the Vanadium playground.
//
// TODO(ivanpi): Port the namespace browser test logic from shell to Go. Add more tests.
func vanadiumPlaygroundTest(ctx *util.Context, testName string, _ ...TestOpt) (_ *TestResult, e error) {
	root, err := util.VanadiumRoot()
	if err != nil {
		return nil, err
	}

	// Initialize the test.
	cleanup, err := initTest(ctx, testName, []string{"web"})
	if err != nil {
		return nil, internalTestError{err, "Init"}
	}
	defer collect.Error(func() error { return cleanup() }, &e)

	playgroundDir := filepath.Join(root, "release", "projects", "playground")
	backendDir := filepath.Join(playgroundDir, "go", "src", "playground")
	clientDir := filepath.Join(playgroundDir, "client")

	// Clean the playground client build.
	if err := ctx.Run().Chdir(clientDir); err != nil {
		return nil, err
	}
	if err := ctx.Run().Command("make", "clean"); err != nil {
		return nil, err
	}

	// Run builder integration test.
	if testResult, err := vanadiumPlaygroundSubtest(ctx, testName, "builder integration", backendDir, "test"); testResult != nil || err != nil {
		return testResult, err
	}

	// Run client embedded example test.
	if testResult, err := vanadiumPlaygroundSubtest(ctx, testName, "client embedded example", clientDir, "test"); testResult != nil || err != nil {
		return testResult, err
	}

	return &TestResult{Status: TestPassed}, nil
}

// Runs specified make target in the specified directory as a test case.
// On success, both return values are nil.
func vanadiumPlaygroundSubtest(ctx *util.Context, testName, caseName, casePath, caseTarget string) (tr *TestResult, err error) {
	if err = ctx.Run().Chdir(casePath); err != nil {
		return
	}
	if err := ctx.Run().TimedCommand(defaultPlaygroundTestTimeout, "make", caseTarget); err != nil {
		if err == runutil.CommandTimedOutErr {
			return &TestResult{
				Status:       TestTimedOut,
				TimeoutValue: defaultPlaygroundTestTimeout,
			}, nil
		} else {
			return nil, internalTestError{err, "Make " + caseTarget}
		}
	}
	return
}
