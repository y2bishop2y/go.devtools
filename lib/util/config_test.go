package util

import (
	"encoding/json"
	"reflect"
	"testing"
)

var (
	goWorkspaces = []string{"test-go-workspace"}
	projectTests = map[string][]string{
		"test-project": []string{"test-test-A", "test-test-group"},
	}
	snapshotLabelTests = map[string][]string{
		"test-snapshot-label": []string{"test-test-A", "test-test-group"},
	}
	testDependencies = map[string][]string{
		"test-test-A": []string{"test-test-B"},
		"test-test-B": []string{"test-test-C"},
	}
	testGroups = map[string][]string{
		"test-test-group": []string{"test-test-B", "test-test-C"},
	}
	vdlWorkspaces = []string{"test-vdl-workspace"}
)

func testConfigAPI(t *testing.T, c *Config) {
	if got, want := c.GoWorkspaces(), goWorkspaces; !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected result: got %v, want %v", got, want)
	}
	if got, want := c.Projects(), []string{"test-project"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected result: got %v, want %v", got, want)
	}
	if got, want := c.ProjectTests("test-project"), []string{"test-test-A", "test-test-B", "test-test-C"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected result: got %v, want %v", got, want)
	}
	if got, want := c.SnapshotLabels(), []string{"test-snapshot-label"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected result: got %v, want %v", got, want)
	}
	if got, want := c.SnapshotLabelTests("test-snapshot-label"), []string{"test-test-A", "test-test-B", "test-test-C"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected result: got %v, want %v", got, want)
	}
	if got, want := c.TestDependencies("test-test-A"), []string{"test-test-B"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected result: got %v, want %v", got, want)
	}
	if got, want := c.TestDependencies("test-test-B"), []string{"test-test-C"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected result: got %v, want %v", got, want)
	}
	if got, want := c.VDLWorkspaces(), vdlWorkspaces; !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected result: got %v, want %v", got, want)
	}
}

func TestConfig(t *testing.T) {
	config := NewConfig(
		GoWorkspaceOpt(goWorkspaces),
		ProjectTestsOpt(projectTests),
		SnapshotLabelTestsOpt(snapshotLabelTests),
		TestDependenciesOpt(testDependencies),
		TestGroupsOpt(testGroups),
		VDLWorkspacesOpt(vdlWorkspaces),
	)

	testConfigAPI(t, config)
}

func TestConfigMarshal(t *testing.T) {
	config := NewConfig(
		GoWorkspaceOpt(goWorkspaces),
		ProjectTestsOpt(projectTests),
		SnapshotLabelTestsOpt(snapshotLabelTests),
		TestDependenciesOpt(testDependencies),
		TestGroupsOpt(testGroups),
		VDLWorkspacesOpt(vdlWorkspaces),
	)

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Marhsall(%v) failed: %v", config, err)
	}
	var config2 Config
	if err := json.Unmarshal(data, &config2); err != nil {
		t.Fatalf("Unmarshall(%v) failed: %v", string(data), err)
	}

	testConfigAPI(t, &config2)
}
