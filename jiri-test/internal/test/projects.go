// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package test

import (
	"path/filepath"
	"time"

	"v.io/jiri/lib/collect"
	"v.io/jiri/lib/project"
	"v.io/jiri/lib/runutil"
	"v.io/jiri/lib/tool"
	"v.io/x/devtools/internal/test"
	"v.io/x/devtools/internal/xunit"
)

const (
	defaultProjectTestTimeout = 10 * time.Minute
)

// runProjectTest is a helper for running project tests.
func runProjectTest(ctx *tool.Context, testName, projectName, target string, env map[string]string, profiles []string) (_ *test.Result, e error) {
	// Initialize the test.
	cleanup, err := initTest(ctx, testName, profiles)
	if err != nil {
		return nil, err
	}
	defer collect.Error(func() error { return cleanup() }, &e)

	// Navigate to project directory.
	root, err := project.V23Root()
	if err != nil {
		return nil, err
	}
	testDir := filepath.Join(root, "release", "projects", projectName)
	if err := ctx.Run().Chdir(testDir); err != nil {
		return nil, err
	}

	// Clean.
	if err := ctx.Run().Command("make", "clean"); err != nil {
		return nil, err
	}

	// Set environment from the env argument map.
	opts := ctx.Run().Opts()
	for k, v := range env {
		opts.Env[k] = v
	}

	// Run the tests.
	if err := ctx.Run().TimedCommandWithOpts(defaultProjectTestTimeout, opts, "make", target); err != nil {
		if err == runutil.CommandTimedOutErr {
			return &test.Result{
				Status:       test.TimedOut,
				TimeoutValue: defaultProjectTestTimeout,
			}, nil
		} else {
			return nil, internalTestError{err, "Make " + target}
		}
	}

	return &test.Result{Status: test.Passed}, nil
}

// vanadiumBrowserTest runs the tests for the Vanadium browser.
func vanadiumBrowserTest(ctx *tool.Context, testName string, _ ...Opt) (*test.Result, error) {
	env := map[string]string{
		"XUNIT_OUTPUT_FILE": xunit.ReportPath(testName),
	}
	return runProjectTest(ctx, testName, "browser", "test", env, []string{"nacl", "nodejs"})
}

// vanadiumBrowserTestWeb runs the ui tests for the Vanadium browser.
func vanadiumBrowserTestWeb(ctx *tool.Context, testName string, _ ...Opt) (*test.Result, error) {
	return runProjectTest(ctx, testName, "browser", "test-ui", nil, []string{"nacl", "nodejs"})
}

// vanadiumChatShellTest runs the tests for the chat shell client.
func vanadiumChatShellTest(ctx *tool.Context, testName string, _ ...Opt) (*test.Result, error) {
	return runProjectTest(ctx, testName, "chat", "test-shell", nil, nil)
}

// vanadiumChatWebTest runs the tests for the chat web client.
func vanadiumChatWebTest(ctx *tool.Context, testName string, _ ...Opt) (*test.Result, error) {
	return runProjectTest(ctx, testName, "chat", "test-web", nil, []string{"nacl", "nodejs"})
}

// vanadiumChatWebUITest runs the ui tests for the chat web client.
func vanadiumChatWebUITest(ctx *tool.Context, testName string, _ ...Opt) (*test.Result, error) {
	return runProjectTest(ctx, testName, "chat", "test-ui", nil, []string{"nacl", "nodejs"})
}

// vanadiumPipe2BrowserTest runs the tests for pipe2browser.
func vanadiumPipe2BrowserTest(ctx *tool.Context, testName string, _ ...Opt) (*test.Result, error) {
	return runProjectTest(ctx, testName, "pipe2browser", "test", nil, []string{"nacl", "nodejs"})
}

// vanadiumReaderTest runs the tests for the reader example application.
func vanadiumReaderTest(ctx *tool.Context, testName string, _ ...Opt) (*test.Result, error) {
	return runProjectTest(ctx, testName, "reader", "test", nil, []string{"nacl", "nodejs"})
}

func vanadiumTravelTest(ctx *tool.Context, testName string, _ ...Opt) (*test.Result, error) {
	return runProjectTest(ctx, testName, "travel", "test", nil, []string{"nodejs"})
}