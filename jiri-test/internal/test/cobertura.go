// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package test

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"v.io/jiri/profiles"
	"v.io/jiri/tool"
	"v.io/jiri/util"
)

type testCoverage struct {
	BranchRate int               `xml:"branch-rate,attr"`
	LineRate   int               `xml:"line-rate,attr"`
	TimeStamp  int64             `xml:"timestamp,attr"`
	Version    string            `xml:"version,attr"`
	Packages   []testCoveragePkg `xml:"packages>package"`
	Sources    []string          `xml:"sources>source"`
	XMLName    xml.Name          `xml:"coverage"`
}

type testCoveragePkg struct {
	Name       string              `xml:"name,attr"`
	BranchRate int                 `xml:"branch-rate,attr"`
	LineRate   int                 `xml:"line-rate,attr"`
	Complexity int                 `xml:"complexity,attr"`
	Classes    []testCoverageClass `xml:"classes>class"`
}

type testCoverageClass struct {
	Name       string               `xml:"name,attr"`
	Filename   string               `xml:"filename,attr"`
	BranchRate int                  `xml:"branch-rate,attr"`
	LineRate   int                  `xml:"line-rate,attr"`
	Complexity int                  `xml:"complexity,attr"`
	Methods    []testCoverageMethod `xml:"methods>method"`
}

type testCoverageMethod struct {
	Name       string             `xml:"name,attr"`
	Signature  string             `xml:"signature,attr"`
	BranchRate int                `xml:"branch-rate,attr"`
	LineRate   int                `xml:"line-rate,attr"`
	Lines      []testCoverageLine `xml:"lines>line"`
}

type testCoverageLine struct {
	Number int `xml:"number,attr"`
	Hits   int `xml:"hits,attr"`
}

// coberturaReportPath returns the path to the cobertura report.
func coberturaReportPath(testName string) string {
	workspace, fileName := os.Getenv("WORKSPACE"), "cobertura_report.xml"
	if workspace == "" {
		return filepath.Join(os.Getenv("HOME"), "tmp", testName, fileName)
	} else {
		return filepath.Join(workspace, fileName)
	}
}

// coverageFromGoTestOutput reads data from the given input, assuming
// it contains the coverage information generated by "go test -cover",
// and returns it as an in-memory data structure.
func coverageFromGoTestOutput(ctx *tool.Context, testOutput io.Reader) (*testCoverage, error) {
	bin, err := util.ThirdPartyBinPath("gocover-cobertura")
	if err != nil {
		return nil, err
	}
	ch, err := profiles.NewConfigHelper(ctx, profiles.DefaultManifestFilename)
	if err != nil {
		return nil, err
	}
	ch.SetGoPath()
	var out bytes.Buffer
	opts := ctx.Run().Opts()
	opts.Stdin = testOutput
	opts.Stdout = &out
	opts.Env = ch.ToMap()
	if err := ctx.Run().CommandWithOpts(opts, bin); err != nil {
		return nil, err
	}
	var coverage testCoverage
	if err := xml.Unmarshal(out.Bytes(), &coverage); err != nil {
		return nil, fmt.Errorf("Unmarshal() failed: %v\n%v", err, out.String())
	}
	return &coverage, nil
}

// createCoberturaReport generates a cobertura report using the given
// coverage information.
func createCoberturaReport(ctx *tool.Context, testName string, data *testCoverage) error {
	bytes, err := xml.MarshalIndent(*data, "", "  ")
	if err != nil {
		return fmt.Errorf("MarshalIndent(%v) failed: %v", *data, err)
	}
	if err := ctx.Run().WriteFile(coberturaReportPath(testName), bytes, os.FileMode(0644)); err != nil {
		return fmt.Errorf("WriteFile(%v) failed: %v", coberturaReportPath(testName), err)
	}
	return nil
}
