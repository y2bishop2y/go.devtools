package testutil

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"veyron.io/tools/lib/util"
)

// veyronVDL checks that all VDL-based Go source files are up-to-date.
func (t *testEnv) veyronVDL(ctx *util.Context, testName string) (*TestResult, error) {
	root, err := util.VeyronRoot()
	if err != nil {
		return nil, err
	}

	// Install the vdl tool.
	opts := t.setTestEnv(ctx.Run().Opts())
	opts.Env["GOPATH"] = filepath.Join(root, "veyron", "go")
	if err := ctx.Run().CommandWithOpts(opts, "go", "install", "veyron.io/veyron/veyron2/vdl/vdl"); err != nil {
		return nil, err
	}

	// Check that "vdl audit --lang=go all" produces no output.
	var out bytes.Buffer
	opts = t.setTestEnv(ctx.Run().Opts())
	opts.Stdout = &out
	opts.Stderr = &out
	venv, err := util.VeyronEnvironment(util.HostPlatform())
	if err != nil {
		return nil, err
	}
	opts.Env["VDLPATH"] = venv.Get("VDLPATH")
	vdl := filepath.Join(root, "veyron", "go", "bin", "vdl")
	err = ctx.Run().CommandWithOpts(opts, vdl, "audit", "--lang=go", "all")
	output := strings.TrimSpace(out.String())
	if err != nil || len(output) != 0 {
		fmt.Fprintf(ctx.Stdout(), "%v\n", output)
		return &TestResult{Status: TestFailed}, nil
	}
	return &TestResult{Status: TestPassed}, nil
}
