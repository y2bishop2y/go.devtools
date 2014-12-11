package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"veyron.io/lib/cmdline"
	"veyron.io/tools/lib/testutil"
	"veyron.io/tools/lib/util"
)

func TestTestProject(t *testing.T) {
	ctx := util.DefaultContext()

	// Setup an instance of veyron universe.
	rootDir, err := ctx.Run().TempDir("", "")
	if err != nil {
		t.Fatalf("TempDir() failed: %v", err)
	}
	defer ctx.Run().RemoveAll(rootDir)
	oldRoot := os.Getenv("VEYRON_ROOT")
	if err := os.Setenv("VEYRON_ROOT", rootDir); err != nil {
		t.Fatalf("%v", err)
	}
	defer os.Setenv("VEYRON_ROOT", oldRoot)

	config := util.CommonConfig{
		ProjectTests: map[string][]string{
			"https://test-project": []string{"ignore-this"},
		},
	}
	createConfig(t, ctx, &config)

	// Check that running the tests for the test project generates
	// the expected output.
	var out bytes.Buffer
	command := cmdline.Command{}
	command.Init(nil, &out, &out)
	if err := runTestProject(&command, []string{"https://test-project"}); err != nil {
		t.Fatalf("%v", err)
	}
	got, want := out.String(), `##### Running test "ignore-this" #####
##### PASSED #####
SUMMARY:
ignore-this PASSED
`
	if got != want {
		t.Fatalf("unexpected output:\ngot\n%v\nwant\n%v", got, want)
	}
}

func TestTestRun(t *testing.T) {
	ctx := util.DefaultContext()

	// Setup an instance of veyron universe.
	rootDir, err := ctx.Run().TempDir("", "")
	if err != nil {
		t.Fatalf("TempDir() failed: %v", err)
	}
	defer ctx.Run().RemoveAll(rootDir)
	oldRoot := os.Getenv("VEYRON_ROOT")
	if err := os.Setenv("VEYRON_ROOT", rootDir); err != nil {
		t.Fatalf("%v", err)
	}
	defer os.Setenv("VEYRON_ROOT", oldRoot)

	// Check that running the test generates the expected output.
	var out bytes.Buffer
	command := cmdline.Command{}
	command.Init(nil, &out, &out)
	if err := runTestRun(&command, []string{"ignore-this"}); err != nil {
		t.Fatalf("%v", err)
	}
	got, want := out.String(), `##### Running test "ignore-this" #####
##### PASSED #####
SUMMARY:
ignore-this PASSED
`
	if got != want {
		t.Fatalf("unexpected output:\ngot\n%v\nwant\n%v", got, want)
	}
}

func TestTestList(t *testing.T) {
	ctx := util.DefaultContext()

	// Setup an instance of veyron universe.
	rootDir, err := ctx.Run().TempDir("", "")
	if err != nil {
		t.Fatalf("TempDir() failed: %v", err)
	}
	defer ctx.Run().RemoveAll(rootDir)
	oldRoot := os.Getenv("VEYRON_ROOT")
	if err := os.Setenv("VEYRON_ROOT", rootDir); err != nil {
		t.Fatalf("%v", err)
	}
	defer os.Setenv("VEYRON_ROOT", oldRoot)

	// Check that listing existing tests generates the expected
	// output.
	var out bytes.Buffer
	command := cmdline.Command{}
	command.Init(nil, &out, &out)
	if err := runTestList(&command, []string{}); err != nil {
		t.Fatalf("%v", err)
	}
	testList, err := testutil.TestList()
	if err != nil {
		t.Fatalf("%v", err)
	}
	if got, want := strings.TrimSpace(out.String()), strings.Join(testList, "\n"); got != want {
		t.Fatalf("unexpected output:\ngot\n%v\nwant\n%v", got, want)
	}
}
