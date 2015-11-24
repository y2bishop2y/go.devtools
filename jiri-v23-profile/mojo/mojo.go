// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// package mojo implements the mojo profile.
package mojo_profile

import (
	"bytes"
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"v.io/jiri/jiri"
	"v.io/jiri/profiles"
	"v.io/x/lib/envvar"
)

const (
	profileName = "mojo"

	// On every green Mojo build (defined as compiling and passing the mojo
	// tests) a set of build artifacts are published to publicly-readable
	// Google cloud storage buckets.
	mojoStorageBucket = "https://storage.googleapis.com/mojo"

	// The mojo devtools repo has tools for running, debugging and testing mojo apps.
	mojoDevtoolsRemote = "https://github.com/domokit/devtools"

	// The mojo_sdk repo is a mirror of github.com/domokit/mojo/mojo/public.
	// It is mirrored for easy consumption.
	mojoSdkRemote = "https://github.com/domokit/mojo_sdk.git"

	// The main mojo repo.  We should not need this, but currently the service
	// mojom files live here and are not mirrored anywhere else.
	// TODO(nlacasse): Once the service mojoms exist elsewhere, remove this and
	// get the service mojoms from wherever they are.
	mojoRemote = "https://github.com/domokit/mojo.git"
)

type versionSpec struct {
	// Version of android platform tools.  See http://tools.android.com/download.
	androidPlatformToolsVersion string

	// The names of the mojo services to install for all targets.
	serviceNames []string

	// The names of additional mojo services to install for android targets.
	serviceNamesAndroid []string

	// The names of additional mojo services to install for linux targets.
	serviceNamesLinux []string

	// The git SHA of the mojo artifacts, including the mojo shell and system
	// thunks to install on android targets.
	// The latest can be found in
	// https://storage.googleapis.com/mojo/shell/android-arm/LATEST.
	buildVersionAndroid string

	// The git SHA of the mojo artifacts, including the mojo shell and system
	// thunks to install on linux targets.
	// The latest can be found in
	// https://storage.googleapis.com/mojo/shell/linux-x64/LATEST.
	buildVersionLinux string

	// The git SHA, branch, or tag of the devtools repo to checkout.
	devtoolsVersion string

	// The git SHA of the mojo network service.  The latest can be found in
	// https://github.com/domokit/mojo/blob/master/mojo/public/tools/NETWORK_SERVICE_VERSION
	networkServiceVersion string

	// The git SHA, branch, or tag of the the mojo_sdk repo to checkout.
	sdkVersion string
}

func init() {
	m := &Manager{
		versionInfo: profiles.NewVersionInfo(profileName, map[string]interface{}{
			"1": &versionSpec{
				serviceNames: []string{
					"authenticating_url_loader_interceptor.mojo",
					"dart_content_handler.mojo",
					"debugger.mojo",
					"kiosk_wm.mojo",
					"tracing.mojo",
				},
				serviceNamesAndroid: []string{},
				serviceNamesLinux: []string{
					"authentication.mojo",
				},
				buildVersionAndroid:         "e2cd09460972dab4d1766153e108457fe5bbaed5",
				buildVersionLinux:           "e2cd09460972dab4d1766153e108457fe5bbaed5",
				devtoolsVersion:             "a264dd5ebdb5508d4e7e432b0ee3dcf6b1fb7160",
				networkServiceVersion:       "0a814ed5512598e595c0ae7975a09d90a7a54e90",
				sdkVersion:                  "b3af6aeeea02c07e7ccb2c672a0ebcda0d6c42b4",
				androidPlatformToolsVersion: "2219198",
			},
		}, "1"),
	}
	profiles.Register(profileName, m)
}

type Manager struct {
	root, mojoRoot, mojoInstDir, androidPlatformToolsDir profiles.RelativePath
	devtoolsDir, sdkDir, shellDir, systemThunksDir       profiles.RelativePath
	versionInfo                                          *profiles.VersionInfo
	spec                                                 versionSpec
	buildVersion                                         string
	platform                                             string
}

func (m Manager) Name() string {
	return profileName
}

func (m Manager) String() string {
	return fmt.Sprintf("%s[%s]", profileName, m.versionInfo.Default())
}

func (m Manager) VersionInfo() *profiles.VersionInfo {
	return m.versionInfo
}

func (m Manager) Info() string {
	return `Downloads pre-built mojo binaries and other assets required for building mojo servcies.`
}

func (m *Manager) AddFlags(flags *flag.FlagSet, action profiles.Action) {
}

func (m *Manager) initForTarget(jirix *jiri.X, root profiles.RelativePath, target *profiles.Target) error {
	if err := m.versionInfo.Lookup(target.Version(), &m.spec); err != nil {
		return err
	}

	// Turn "amd64" architecture string into "x64" to match mojo's usage.
	mojoArch := target.Arch()
	if mojoArch == "amd64" {
		mojoArch = "x64"
	}
	m.platform = target.OS() + "-" + mojoArch

	if m.platform != "linux-x64" && m.platform != "android-arm" {
		return fmt.Errorf("only amd64-linux and arm-android targets are supported for mojo profile")
	}

	m.buildVersion = m.spec.buildVersionLinux
	if m.platform == "android-arm" {
		m.buildVersion = m.spec.buildVersionAndroid
	}

	m.root = root
	m.mojoRoot = root.Join("mojo")

	// devtools and mojo sdk are not architecture-dependant, so they can go in
	// mojoRoot.
	m.devtoolsDir = m.mojoRoot.Join("devtools", m.spec.devtoolsVersion)
	m.sdkDir = m.mojoRoot.Join("mojo_sdk", m.spec.sdkVersion)

	m.mojoInstDir = m.mojoRoot.Join(target.TargetSpecificDirname())
	m.androidPlatformToolsDir = m.mojoInstDir.Join("platform-tools", m.spec.androidPlatformToolsVersion)
	m.shellDir = m.mojoInstDir.Join("mojo_shell", m.buildVersion)
	m.systemThunksDir = m.mojoInstDir.Join("system_thunks", m.buildVersion)

	if jirix.Verbose() {
		fmt.Fprintf(jirix.Stdout(), "Installation Directories for: %s\n", target)
		fmt.Fprintf(jirix.Stdout(), "mojo installation dir: %s\n", m.mojoInstDir)
		fmt.Fprintf(jirix.Stdout(), "devtools: %s\n", m.devtoolsDir)
		fmt.Fprintf(jirix.Stdout(), "sdk: %s\n", m.sdkDir)
		fmt.Fprintf(jirix.Stdout(), "shell: %s\n", m.shellDir)
		fmt.Fprintf(jirix.Stdout(), "system thunks: %s\n", m.systemThunksDir)
		fmt.Fprintf(jirix.Stdout(), "android platform tools: %s\n", m.androidPlatformToolsDir)
	}
	return nil
}

func (m *Manager) Install(jirix *jiri.X, root profiles.RelativePath, target profiles.Target) error {
	if err := m.initForTarget(jirix, root, &target); err != nil {
		return err
	}

	seq := jirix.NewSeq()
	seq.MkdirAll(m.mojoInstDir.Expand(), profiles.DefaultDirPerm).
		Call(func() error { return m.installMojoDevtools(jirix, m.devtoolsDir.Expand()) }, "install mojo devtools").
		Call(func() error { return m.installMojoSdk(jirix, m.sdkDir.Expand()) }, "install mojo SDK").
		Call(func() error { return m.installMojoShellAndServices(jirix, m.shellDir.Expand()) }, "install mojo shell and services").
		Call(func() error { return m.installMojoSystemThunks(jirix, m.systemThunksDir.Expand()) }, "install mojo system thunks")

	target.Env.Vars = envvar.MergeSlices(target.Env.Vars, []string{
		"CGO_CFLAGS=-I" + m.sdkDir.Join("src").Expand(),
		"CGO_CXXFLAGS=-I" + m.sdkDir.Join("src").Expand(),
		"CGO_LDFLAGS=-L" + m.systemThunksDir.Expand() + " -lsystem_thunks",
		"GOPATH=" + m.sdkDir.Expand() + ":" + m.sdkDir.Join("gen", "go").Expand(),
		"MOJO_DEVTOOLS=" + m.devtoolsDir.Expand(),
		"MOJO_SDK=" + m.sdkDir.Expand(),
		"MOJO_SHELL=" + m.shellDir.Join("mojo_shell").Expand(),
		"MOJO_SERVICES=" + m.shellDir.Expand(),
		"MOJO_SYSTEM_THUNKS=" + m.systemThunksDir.Join("libsystem_thunks.a").Expand(),
	})

	if m.platform == "android-arm" {
		seq.Call(func() error { return m.installAndroidPlatformTools(jirix, m.androidPlatformToolsDir.Expand()) }, "install android platform tools")
		target.Env.Vars = envvar.MergeSlices(target.Env.Vars, []string{
			"ANDROID_PLATFORM_TOOLS=" + m.androidPlatformToolsDir.Expand(),
			"MOJO_SHELL=" + m.shellDir.Join("MojoShell.apk").Expand(),
		})
	}

	if err := seq.Done(); err != nil {
		return err
	}

	if profiles.SchemaVersion() >= 4 {
		target.InstallationDir = m.mojoInstDir.String()
		profiles.InstallProfile(profileName, m.mojoRoot.String())
	} else {
		target.InstallationDir = m.mojoInstDir.Expand()
		profiles.InstallProfile(profileName, m.mojoRoot.Expand())
	}

	return profiles.AddProfileTarget(profileName, target)
}

// installAndroidPlatformTools installs the android platform tools in outDir.
func (m *Manager) installAndroidPlatformTools(jirix *jiri.X, outDir string) error {
	tmpDir, err := jirix.NewSeq().TempDir("", "")
	if err != nil {
		return err
	}
	defer jirix.NewSeq().RemoveAll(tmpDir)

	fn := func() error {
		androidPlatformToolsZipFile := filepath.Join(tmpDir, "platform-tools.zip")
		return jirix.NewSeq().
			Call(func() error {
			return profiles.Fetch(jirix, androidPlatformToolsZipFile, androidPlatformToolsUrl(m.spec.androidPlatformToolsVersion))
		}, "fetch android platform tools").
			Call(func() error { return profiles.Unzip(jirix, androidPlatformToolsZipFile, tmpDir) }, "unzip android platform tools").
			MkdirAll(filepath.Dir(outDir), profiles.DefaultDirPerm).
			Rename(filepath.Join(tmpDir, "platform-tools"), outDir).
			Done()
	}
	return profiles.AtomicAction(jirix, fn, outDir, "Install Android Platform Tools")
}

// installMojoNetworkService installs network_services.mojo into outDir.
func (m *Manager) installMojoNetworkService(jirix *jiri.X, outDir string) error {
	tmpDir, err := jirix.NewSeq().TempDir("", "")
	if err != nil {
		return err
	}
	defer jirix.NewSeq().RemoveAll(tmpDir)

	networkServiceUrl := mojoNetworkServiceUrl(m.platform, m.spec.networkServiceVersion)
	networkServiceZipFile := filepath.Join(tmpDir, "network_service.mojo.zip")
	tmpFile := filepath.Join(tmpDir, "network_service.mojo")
	outFile := filepath.Join(outDir, "network_service.mojo")

	return jirix.NewSeq().
		Call(func() error { return profiles.Fetch(jirix, networkServiceZipFile, networkServiceUrl) }, "fetch %s", networkServiceUrl).
		Call(func() error { return profiles.Unzip(jirix, networkServiceZipFile, tmpDir) }, "unzip network service").
		MkdirAll(filepath.Dir(outDir), profiles.DefaultDirPerm).
		Rename(tmpFile, outFile).
		Done()
}

// installMojoDevtools clones the mojo devtools repo into outDir.
func (m *Manager) installMojoDevtools(jirix *jiri.X, outDir string) error {
	fn := func() error {
		return jirix.NewSeq().
			MkdirAll(outDir, profiles.DefaultDirPerm).
			Pushd(outDir).
			Call(func() error { return jirix.Git().Clone(mojoDevtoolsRemote, outDir) }, "git clone %s", mojoDevtoolsRemote).
			Call(func() error { return jirix.Git().Reset(m.spec.devtoolsVersion) }, "git reset --hard %s", m.spec.devtoolsVersion).
			Popd().
			Done()
	}
	return profiles.AtomicAction(jirix, fn, outDir, "Install Mojo devtools")
}

// installMojoSdk clones the mojo_sdk repo into outDir/src/mojo/public.  It
// also generates .mojom.go files from the .mojom files.
func (m *Manager) installMojoSdk(jirix *jiri.X, outDir string) error {
	fn := func() error {
		seq := jirix.NewSeq()
		// TODO(nlacasse): At some point Mojo needs to change the structure of
		// their repo so that go packages occur with correct paths. Until then
		// we'll clone into src/mojo/public so that go import paths work.
		repoDst := filepath.Join(outDir, "src", "mojo", "public")
		seq.
			MkdirAll(repoDst, profiles.DefaultDirPerm).
			Pushd(repoDst).
			Call(func() error { return jirix.Git().Clone(mojoSdkRemote, repoDst) }, "git clone %s", mojoSdkRemote).
			Call(func() error { return jirix.Git().Reset(m.spec.sdkVersion) }, "git reset --hard %s", m.spec.sdkVersion).
			Popd()

		// Download the authentication and network service mojom files.
		// TODO(nlacasse): This is a HACK.  The service mojom files are not
		// published anywhere yet, so we get them from the main mojo repo,
		// which we should not need to do.  Once they are published someplace
		// else, get them from there.
		tmpMojoCheckout, err := jirix.NewSeq().TempDir("", "")
		if err != nil {
			return err
		}
		defer jirix.NewSeq().RemoveAll(tmpMojoCheckout)

		seq.
			Pushd(tmpMojoCheckout).
			Call(func() error { return jirix.Git().Clone(mojoRemote, tmpMojoCheckout) }, "git clone %s", mojoRemote).
			Call(func() error { return jirix.Git().Reset(m.buildVersion) }, "git reset --hard %s", m.buildVersion).
			Popd()

		servicesSrc := filepath.Join(tmpMojoCheckout, "mojo", "services")
		servicesDst := filepath.Join(outDir, "src", "mojo", "services")
		seq.Rename(servicesSrc, servicesDst)

		// Find all .mojom files.
		var mojomFilesBuffer bytes.Buffer
		if err := jirix.NewSeq().Capture(&mojomFilesBuffer, nil).Last("find", outDir, "-name", "*.mojom"); err != nil {
			return err
		}
		mojomFiles := strings.Split(mojomFilesBuffer.String(), "\n")

		// Generate the mojom.go files from all mojom files.
		seq.Pushd(filepath.Join(outDir, "src"))
		genMojomTool := filepath.Join(outDir, "src", "mojo", "public", "tools", "bindings", "mojom_bindings_generator.py")
		for _, mojomFile := range mojomFiles {
			trimmedFile := strings.TrimSpace(mojomFile)
			if trimmedFile == "" {
				continue
			}
			seq.Run(genMojomTool,
				"--use_bundled_pylibs",
				"-g", "go",
				"-o", filepath.Join("..", "gen"),
				"-I", ".",
				"-I", servicesDst,
				trimmedFile)
		}
		seq.Popd()

		return seq.Done()
	}

	return profiles.AtomicAction(jirix, fn, outDir, "Clone Mojo SDK repository")
}

// installMojoShellAndServices installs the mojo shell and all services into outDir.
func (m *Manager) installMojoShellAndServices(jirix *jiri.X, outDir string) error {
	tmpDir, err := jirix.NewSeq().TempDir("", "")
	if err != nil {
		return err
	}
	defer jirix.NewSeq().RemoveAll(tmpDir)

	fn := func() error {
		seq := jirix.NewSeq()
		seq.MkdirAll(outDir, profiles.DefaultDirPerm)

		// Install mojo shell.
		url := mojoShellUrl(m.platform, m.buildVersion)
		mojoShellZipFile := filepath.Join(tmpDir, "mojo_shell.zip")
		seq.
			Call(func() error { return profiles.Fetch(jirix, mojoShellZipFile, url) }, "fetch %s", url).
			Call(func() error { return profiles.Unzip(jirix, mojoShellZipFile, tmpDir) }, "unzip %s", mojoShellZipFile)

		files := []string{"mojo_shell", "mojo_shell_child"}
		if m.platform == "android-arm" {
			// On android, mojo shell is called "MojoShell.apk".
			files = []string{"MojoShell.apk"}
		}
		for _, file := range files {
			tmpFile := filepath.Join(tmpDir, file)
			outFile := filepath.Join(outDir, file)
			seq.Rename(tmpFile, outFile)
		}

		// Install the network services.
		seq.Call(func() error {
			return m.installMojoNetworkService(jirix, outDir)
		}, "install mojo network service")

		// Install all other services.
		serviceNames := m.spec.serviceNames
		if m.platform == "android-arm" {
			serviceNames = append(serviceNames, m.spec.serviceNamesAndroid...)
		}
		if m.platform == "linux-x64" {
			serviceNames = append(serviceNames, m.spec.serviceNamesLinux...)
		}
		for _, serviceName := range serviceNames {
			outFile := filepath.Join(outDir, serviceName)
			serviceUrl := mojoServiceUrl(m.platform, serviceName, m.buildVersion)
			seq.Call(func() error { return profiles.Fetch(jirix, outFile, serviceUrl) }, "fetch %s", serviceUrl)
		}
		return seq.Done()
	}

	return profiles.AtomicAction(jirix, fn, outDir, "install mojo_shell")
}

// installMojoSystemThunks installs the mojo system thunks lib into outDir.
func (m *Manager) installMojoSystemThunks(jirix *jiri.X, outDir string) error {
	fn := func() error {
		outFile := filepath.Join(outDir, "libsystem_thunks.a")
		return jirix.NewSeq().MkdirAll(outDir, profiles.DefaultDirPerm).
			Call(func() error {
			return profiles.Fetch(jirix, outFile, mojoSystemThunksUrl(m.platform, m.buildVersion))
		}, "fetch mojo system thunks").Done()
	}
	return profiles.AtomicAction(jirix, fn, outDir, "Download Mojo system thunks")
}

func (m *Manager) Uninstall(jirix *jiri.X, root profiles.RelativePath, target profiles.Target) error {
	// TODO(nlacasse): What should we do with all the installed artifacts?
	// They could be used by other profile versions, so deleting them does not
	// make sense.  Should we check that they are only used by this profile
	// before deleting?
	profiles.RemoveProfileTarget(profileName, target)
	return nil
}

// androidPlatformToolsUrl returns the url of the android platform tools zip
// file for the given version.
func androidPlatformToolsUrl(version string) string {
	return fmt.Sprintf("http://tools.android.com/download/sdk-repo-linux-platform-tools-%s.zip", version)
}

// mojoNetworkServiceUrl returns the url for the network service for the given
// platform and git revision.
func mojoNetworkServiceUrl(platform, gitRevision string) string {
	return mojoStorageBucket + fmt.Sprintf("/network_service/%s/%s/network_service.mojo.zip", gitRevision, platform)
}

// mojoServiceUrl returns the url for the service for the given platform, name,
// and git revision.
func mojoServiceUrl(platform, name, gitRevision string) string {
	return mojoStorageBucket + fmt.Sprintf("/services/%s/%s/%s", platform, gitRevision, name)
}

// mojoShellUrl returns the url for the mojo shell binary given platform and
// git revision.
func mojoShellUrl(platform, gitRevision string) string {
	return mojoStorageBucket + fmt.Sprintf("/shell/%s/%s.zip", gitRevision, platform)
}

// mojoSystemThunksUrl returns the url for the mojo system thunks binary for the
// given platform and git revision.
func mojoSystemThunksUrl(platform, gitRevision string) string {
	return mojoStorageBucket + fmt.Sprintf("/system_thunks/%s/%s/libsystem_thunks.a", platform, gitRevision)
}