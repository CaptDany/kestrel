package truenas_test

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// AppYAML models truenas/app.yaml
type AppYAML struct {
	Annotations    map[string]string `yaml:"annotations"`
	AppVersion     string            `yaml:"app_version"`
	LibVersion     string            `yaml:"lib_version"`
	LibVersionHash string            `yaml:"lib_version_hash"`
	Name           string            `yaml:"name"`
	Train          string            `yaml:"train"`
	RunAsContext   []RunAsEntry      `yaml:"run_as_context"`
}

type RunAsEntry struct {
	Description string `yaml:"description"`
	GID         int    `yaml:"gid"`
	GroupName   string `yaml:"group_name"`
	UID         int    `yaml:"uid"`
	UserName    string `yaml:"user_name"`
}

func readAppYAML(t *testing.T) AppYAML {
	t.Helper()
	data, err := os.ReadFile(absPath(t, "app.yaml"))
	if err != nil {
		t.Fatalf("read app.yaml: %v", err)
	}
	var a AppYAML
	if err := yaml.Unmarshal(data, &a); err != nil {
		t.Fatalf("unmarshal app.yaml: %v", err)
	}
	return a
}

// ──────────────────────────────────────────────
// Functional tests
// ──────────────────────────────────────────────

func TestAppFunctionalLibVersionHash(t *testing.T) {
	a := readAppYAML(t)
	if a.LibVersionHash == "" {
		t.Error("F.1: lib_version_hash must not be empty")
	}
	// SHA256 hex hashes are 64 characters
	if len(a.LibVersionHash) != 64 {
		t.Errorf("F.1: lib_version_hash length %d, expected 64", len(a.LibVersionHash))
	}
	// Must be valid hex
	for _, c := range a.LibVersionHash {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			t.Errorf("F.1: lib_version_hash contains invalid hex char %c", c)
			break
		}
	}
}

func TestAppFunctionalRunAsContextCount(t *testing.T) {
	a := readAppYAML(t)
	// After removing scraper + permissions, should have 1 entry (main container only, like Jellyfin)
	if len(a.RunAsContext) != 1 {
		t.Errorf("F.2: Expected 1 run_as_context entry, got %d", len(a.RunAsContext))
	}
}

func TestAppFunctionalRunAsContextNoScraper(t *testing.T) {
	a := readAppYAML(t)
	for _, entry := range a.RunAsContext {
		if strings.Contains(strings.ToLower(entry.Description), "scraper") {
			t.Error("F.3: run_as_context must not contain scraper entry")
		}
	}
}

func TestAppFunctionalRunAsContextNoPermissions(t *testing.T) {
	a := readAppYAML(t)
	for _, entry := range a.RunAsContext {
		if strings.Contains(strings.ToLower(entry.Description), "permissions") {
			t.Error("F.4: run_as_context must not contain permissions entry")
		}
	}
}

func TestAppFunctionalMainContainerEntry(t *testing.T) {
	a := readAppYAML(t)
	if len(a.RunAsContext) < 1 {
		t.Skip("no entries to check")
	}
	entry := a.RunAsContext[0]
	if entry.UID != 568 || entry.GID != 568 {
		t.Errorf("F.5: Main run_as expected UID/GID 568, got %d/%d", entry.UID, entry.GID)
	}
}

// ──────────────────────────────────────────────
// Structural / Non-functional tests
// ──────────────────────────────────────────────

func TestAppStructureAnnotations(t *testing.T) {
	a := readAppYAML(t)
	requiredAnns := []string{"min_scale_version"}
	for _, ann := range requiredAnns {
		if _, ok := a.Annotations[ann]; !ok {
			t.Errorf("S.1: Missing annotation %q", ann)
		}
	}
}

func TestAppStructureAppVersionFormat(t *testing.T) {
	a := readAppYAML(t)
	if a.AppVersion == "" {
		t.Error("S.2: app_version must not be empty")
	}
	if !strings.Contains(a.AppVersion, ".") {
		t.Errorf("S.2: app_version has no dots: %q", a.AppVersion)
	}
}

func TestAppStructureLibVersion(t *testing.T) {
	a := readAppYAML(t)
	if a.LibVersion == "" {
		t.Error("S.3: lib_version must not be empty")
	}
}

// ──────────────────────────────────────────────
// Random / Fuzz
// ──────────────────────────────────────────────

func TestAppRandomPartition(t *testing.T) {
	// P.1: Delete random optional fields, verify critical fields survive
	critical := map[string]bool{
		"lib_version":       true,
		"app_version":       true,
		"lib_version_hash":  true,
		"name":              true,
		"train":             true,
		"version":           true,
	}

	for _, frac := range []float64{0.0, 0.25, 0.50} {
		t.Run(fmt.Sprintf("%.0fpct", frac*100), func(t *testing.T) {
			data, err := os.ReadFile(absPath(t, "app.yaml"))
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			var raw map[string]any
			if err := yaml.Unmarshal(data, &raw); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			// Delete random non-critical keys
			optKeys := make([]string, 0)
			for k := range raw {
				if !critical[k] {
					optKeys = append(optKeys, k)
				}
			}
			rand.Shuffle(len(optKeys), func(i, j int) {
				optKeys[i], optKeys[j] = optKeys[j], optKeys[i]
			})
			count := int(float64(len(optKeys)) * frac)
			for i := 0; i < count; i++ {
				delete(raw, optKeys[i])
			}
			for k := range critical {
				if _, ok := raw[k]; !ok {
					t.Errorf("P.1: Critical field %q lost after %.0f%% deletion", k, frac*100)
				}
			}
		})
	}
}

func TestAppRandomFuzz(t *testing.T) {
	a := readAppYAML(t)
	data, err := yaml.Marshal(a)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for i := 0; i < 5; i++ {
		m["x-fuzz-"+randomString(6)] = randomString(rand.Intn(16) + 1)
	}
	out, err := yaml.Marshal(m)
	if err != nil {
		t.Fatalf("fuzz marshal: %v", err)
	}
	var round AppYAML
	if err := yaml.Unmarshal(out, &round); err != nil {
		t.Fatalf("fuzz unmarshal: %v", err)
	}
}

// ──────────────────────────────────────────────
// Performance
// ──────────────────────────────────────────────

func TestAppParsePerformance(t *testing.T) {
	file := absPath(t, "app.yaml")
	for i := 0; i < 100; i++ {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		var a AppYAML
		if err := yaml.Unmarshal(data, &a); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
	}
}

// ──────────────────────────────────────────────
// Integration / Smoke
// ──────────────────────────────────────────────

func TestAppIntegrationSmokeChain(t *testing.T) {
	a := readAppYAML(t)
	if a.AppVersion == "" {
		t.Error("I.1: app_version empty")
	}
	if a.LibVersion == "" {
		t.Error("I.1: lib_version empty")
	}
	if a.Name != "kestrel" {
		t.Errorf("I.1: name = %q, want %q", a.Name, "kestrel")
	}
	if a.Train != "community" {
		t.Errorf("I.1: train = %q, want %q", a.Train, "community")
	}
	if a.RunAsContext[0].UID != 568 {
		t.Errorf("I.1: expected UID 568, got %d", a.RunAsContext[0].UID)
	}
}

// ──────────────────────────────────────────────
// Utilities
// ──────────────────────────────────────────────

func absPath(t *testing.T, parts ...string) string {
	t.Helper()
	cwd, _ := os.Getwd()
	if strings.HasSuffix(cwd, "truenas") {
		return filepath.Join(append([]string{cwd}, parts...)...)
	}
	return filepath.Join(append([]string{cwd, "truenas"}, parts...)...)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
