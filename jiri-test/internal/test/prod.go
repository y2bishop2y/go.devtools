// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"time"

	"v.io/jiri/lib/collect"
	"v.io/jiri/lib/project"
	"v.io/jiri/lib/retry"
	"v.io/jiri/lib/tool"
	"v.io/x/devtools/internal/test"
	"v.io/x/devtools/internal/xunit"
)

// generateXUnitTestSuite generates an xUnit test suite that
// encapsulates the given input.
func generateXUnitTestSuite(ctx *tool.Context, failure *xunit.Failure, pkg string, duration time.Duration) *xunit.TestSuite {
	// Generate an xUnit test suite describing the result.
	s := xunit.TestSuite{Name: pkg}
	c := xunit.TestCase{
		Classname: pkg,
		Name:      "Test",
		Time:      fmt.Sprintf("%.2f", duration.Seconds()),
	}
	if failure != nil {
		fmt.Fprintf(ctx.Stdout(), "%s ... failed\n%v\n", pkg, failure.Data)
		c.Failures = append(c.Failures, *failure)
		s.Failures++
	} else {
		fmt.Fprintf(ctx.Stdout(), "%s ... ok\n", pkg)
	}
	s.Tests++
	s.Cases = append(s.Cases, c)
	return &s
}

// testSingleProdService test the given production service.
func testSingleProdService(ctx *tool.Context, vroot, principalDir string, service prodService) *xunit.TestSuite {
	bin := filepath.Join(vroot, "release", "go", "bin", "vrpc")
	var out bytes.Buffer
	opts := ctx.Run().Opts()
	opts.Stdout = &out
	opts.Stderr = &out
	start := time.Now()
	args := []string{}
	if principalDir != "" {
		args = append(args, "--v23.credentials", principalDir)
	}
	args = append(args, "signature", "--show-reserved")
	if principalDir == "" {
		args = append(args, "--insecure")
	}
	args = append(args, service.objectName)
	if err := ctx.Run().TimedCommandWithOpts(test.DefaultTimeout, opts, bin, args...); err != nil {
		return generateXUnitTestSuite(ctx, &xunit.Failure{Message: "vrpc", Data: out.String()}, service.name, time.Now().Sub(start))
	}
	if !service.regexp.Match(out.Bytes()) {
		fmt.Fprintf(ctx.Stderr(), "couldn't match regexp %q in output:\n%v\n", service.regexp, out.String())
		return generateXUnitTestSuite(ctx, &xunit.Failure{Message: "vrpc", Data: "mismatching signature"}, service.name, time.Now().Sub(start))
	}
	return generateXUnitTestSuite(ctx, nil, service.name, time.Now().Sub(start))
}

type prodService struct {
	name       string         // Name to use for the test description
	objectName string         // Object name of the service to connect to
	regexp     *regexp.Regexp // Regexp that should match the signature output
}

// vanadiumProdServicesTest runs a test of vanadium production services.
func vanadiumProdServicesTest(ctx *tool.Context, testName string, opts ...Opt) (_ *test.Result, e error) {
	// Initialize the test.
	cleanup, err := initTest(ctx, testName, nil)
	if err != nil {
		return nil, internalTestError{err, "Init"}
	}
	defer collect.Error(func() error { return cleanup() }, &e)

	vroot, err := project.V23Root()
	if err != nil {
		return nil, err
	}

	// Install the vrpc tool.
	if err := ctx.Run().Command("v23", "go", "install", "v.io/x/ref/cmd/vrpc"); err != nil {
		return nil, internalTestError{err, "Install VRPC"}
	}
	// Install the principal tool.
	if err := ctx.Run().Command("v23", "go", "install", "v.io/x/ref/cmd/principal"); err != nil {
		return nil, internalTestError{err, "Install Principal"}
	}
	tmpdir, err := ctx.Run().TempDir("", "prod-services-test")
	if err != nil {
		return nil, internalTestError{err, "Create temporary directory"}
	}
	defer collect.Error(func() error { return ctx.Run().RemoveAll(tmpdir) }, &e)

	blessingRoot, namespaceRoot := getServiceOpts(opts)
	allPassed, suites := true, []xunit.TestSuite{}

	// Fetch the "root" blessing that all services are blessed by.
	suite, pubkey, blessingNames := testIdentityProviderHTTP(ctx, blessingRoot)
	suites = append(suites, *suite)

	if suite.Failures == 0 {
		// Setup a principal that will be used by testAllProdServices and will
		// recognize the blessings of the prod services.
		principalDir, err := setupPrincipal(ctx, vroot, tmpdir, pubkey, blessingNames)
		if err != nil {
			return nil, err
		}
		for _, suite := range testAllProdServices(ctx, vroot, principalDir, namespaceRoot) {
			allPassed = allPassed && (suite.Failures == 0)
			suites = append(suites, *suite)
		}
	}

	// Create the xUnit report.
	if err := xunit.CreateReport(ctx, testName, suites); err != nil {
		return nil, err
	}
	for _, suite := range suites {
		if suite.Failures > 0 {
			// At least one test failed:
			return &test.Result{Status: test.Failed}, nil
		}
	}
	return &test.Result{Status: test.Passed}, nil
}

func testAllProdServices(ctx *tool.Context, vroot, principalDir, namespaceRoot string) []*xunit.TestSuite {
	services := []prodService{
		prodService{
			name:       "mounttable",
			objectName: namespaceRoot,
			regexp:     regexp.MustCompile(`MountTable[[:space:]]+interface`),
		},
		prodService{
			name:       "application repository",
			objectName: namespaceRoot + "/applications",
			regexp:     regexp.MustCompile(`Application[[:space:]]+interface`),
		},
		prodService{
			name:       "binary repository",
			objectName: namespaceRoot + "/binaries",
			regexp:     regexp.MustCompile(`Binary[[:space:]]+interface`),
		},
		prodService{
			name:       "macaroon service",
			objectName: namespaceRoot + "/identity/dev.v.io/u/macaroon",
			regexp:     regexp.MustCompile(`MacaroonBlesser[[:space:]]+interface`),
		},
		prodService{
			name:       "google identity service",
			objectName: namespaceRoot + "/identity/dev.v.io/u/google",
			regexp:     regexp.MustCompile(`OAuthBlesser[[:space:]]+interface`),
		},
		prodService{
			objectName: namespaceRoot + "/identity/dev.v.io/u/discharger",
			name:       "binary discharger",
			regexp:     regexp.MustCompile(`Discharger[[:space:]]+interface`),
		},
		prodService{
			objectName: namespaceRoot + "/proxy-mon/__debug",
			name:       "proxy service",
			// We just check that the returned signature has the __Reserved interface since
			// proxy-mon doesn't implement any other services.
			regexp: regexp.MustCompile(`__Reserved[[:space:]]+interface`),
		},
	}

	var suites []*xunit.TestSuite
	for _, service := range services {
		suites = append(suites, testSingleProdService(ctx, vroot, principalDir, service))
	}
	return suites
}

// testIdentityProviderHTTP tests that the identity provider's HTTP server is
// up and running and also fetches the set of blessing names that the provider
// claims to be authoritative on and the public key (encoded) used by that
// identity provider to sign certificates for blessings.
//
// PARANOIA ALERT:
// This function is subject to man-in-the-middle attacks because it does not
// verify the TLS certificates presented by the server. This does open the
// door for an attack where a parallel universe of services could be setup
// and fool this production services test into thinking all services are
// up and running when they may not be.
//
// The attacker in this case will have to be able to mess with the routing
// tables on the machine running this test, or the network routes of routers
// used by the machine, or mess up DNS entries.
func testIdentityProviderHTTP(ctx *tool.Context, blessingRoot string) (suite *xunit.TestSuite, publickey string, blessingNames []string) {
	url := fmt.Sprintf("https://%s/auth/blessing-root", blessingRoot)
	var response struct {
		Names     []string `json:"names"`
		PublicKey string   `json:"publicKey"`
	}
	var resp *http.Response
	var err error
	var start time.Time
	fn := func() error {
		start = time.Now()
		resp, err = http.Get(url)
		return err
	}
	if err = retry.Function(ctx, fn); err == nil {
		defer resp.Body.Close()
		err = json.NewDecoder(resp.Body).Decode(&response)
	}
	var failure *xunit.Failure
	if err != nil {
		failure = &xunit.Failure{Message: "identityd HTTP", Data: err.Error()}
	}
	return generateXUnitTestSuite(ctx, failure, url, time.Now().Sub(start)), response.PublicKey, response.Names
}

func setupPrincipal(ctx *tool.Context, vroot, tmpdir, pubkey string, blessingNames []string) (string, error) {
	dir := filepath.Join(tmpdir, "credentials")
	bin := filepath.Join(vroot, "release", "go", "bin", "principal")
	if err := ctx.Run().TimedCommand(test.DefaultTimeout, bin, "create", dir, "prod-services-tester"); err != nil {
		fmt.Fprintf(ctx.Stderr(), "principal create failed: %v\n", err)
		return "", err
	}
	for _, name := range blessingNames {
		if err := ctx.Run().TimedCommand(test.DefaultTimeout, bin, "--v23.credentials", dir, "recognize", name, pubkey); err != nil {
			fmt.Fprintf(ctx.Stderr(), "principal recognize %v %v failed: %v\n", name, pubkey, err)
			return "", err
		}
	}
	return dir, nil
}

// getServiceOpts extracts blessing root and namespace root from the
// given Opts.
func getServiceOpts(opts []Opt) (string, string) {
	blessingRoot := "dev.v.io"
	namespaceRoot := "/ns.dev.v.io:8101"
	for _, opt := range opts {
		switch v := opt.(type) {
		case BlessingsRootOpt:
			blessingRoot = string(v)
		case NamespaceRootOpt:
			namespaceRoot = string(v)
		}
	}
	return blessingRoot, namespaceRoot
}