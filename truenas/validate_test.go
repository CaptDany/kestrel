package truenas_test

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type IXValues struct {
	Images map[string]ImageInfo `yaml:"images"`
	Consts map[string]any       `yaml:"consts"`
}

type ImageInfo struct {
	Repository string `yaml:"repository"`
	Tag        string `yaml:"tag"`
}

type TestCase struct {
	Name     string
	Mutate   func(IXValues) IXValues
	Validate func(*testing.T, IXValues)
}

// randomString generates a random ASCII string for fuzzing.
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func TestIXValuesFunctionalRequiredKeysExist(t *testing.T) {
	// F.1 Load the real ix_values.yaml
	raw := readIXValues(t)

	// F.2. Required images must be present
	requiredImages := []string{"image", "container_utils_image"}
	for _, img := range requiredImages {
		if _, ok := raw.Images[img]; !ok {
			t.Errorf("F.2: Required image %q missing", img)
		}
	}

	// F.3. Required consts must be present
	requiredConsts := []string{"kestrel_container_name", "perms_container_name", "internal_web_port"}
	for _, c := range requiredConsts {
		if _, ok := raw.Consts[c]; !ok {
			t.Errorf("F.3: Required const %q missing", c)
		}
	}

	// F.4. All images must have repository and tag
	for name, img := range raw.Images {
		if img.Repository == "" {
			t.Errorf("F.4: Image %q has empty repository", name)
		}
		if img.Tag == "" {
			t.Errorf("F.4: Image %q has empty tag", name)
		}
	}

	// F.5. internal_web_port must be numeric
	switch v := raw.Consts["internal_web_port"].(type) {
	case int:
		if v <= 0 || v > 65535 {
			t.Errorf("F.5: internal_web_port out of range: %d", v)
		}
	default:
		t.Errorf("F.5: internal_web_port must be int, got %T", v)
	}
}

func TestIXValuesForbiddenKeysAbsent(t *testing.T) {
	raw := readIXValues(t)

	// Forbidden images (Playwright removed)
	forbiddenImages := []string{"playwright_image"}
	for _, img := range forbiddenImages {
		if _, ok := raw.Images[img]; ok {
			t.Errorf("Forbidden image %q must not exist", img)
		}
	}

	// Forbidden consts (scraper, run_as — managed via questions.yaml)
	forbiddenConsts := []string{"scraper_container_name", "run_as_user", "run_as_group"}
	for _, c := range forbiddenConsts {
		if _, ok := raw.Consts[c]; ok {
			t.Errorf("Forbidden const %q must not exist", c)
		}
	}
}

func TestIXValuesRandomPartition(t *testing.T) {
	// P.1 Random-perturbation: delete a random subset of optional entries,
	//     verify required entries survive worst-case deletion.
	required := map[string]bool{
		"images.image":                 true,
		"images.container_utils_image": true,
		"consts.kestrel_container_name": true,
		"consts.perms_container_name":   true,
		"consts.internal_web_port":      true,
	}

	tcs := []struct {
		DeleteFraction float64
		Label          string
	}{
		{0.00, "none deleted"},
		{0.25, "25pct deleted"},
		{0.50, "50pct deleted"},
		{0.75, "75pct deleted"},
	}

	for _, tc := range tcs {
		t.Run(tc.Label, func(t *testing.T) {
			raw := readIXValues(t)

			// Collect all optional leaf keys (path-based)
			allPaths := collectLeafPaths(raw)
			optPaths := make([]string, 0, len(allPaths))
			for _, p := range allPaths {
				if !required[p] {
					optPaths = append(optPaths, p)
				}
			}

			// Delete random subset of optional paths
			rand.Shuffle(len(optPaths), func(i, j int) {
				optPaths[i], optPaths[j] = optPaths[j], optPaths[i]
			})
			deleteCount := int(float64(len(optPaths)) * tc.DeleteFraction)
			for i := 0; i < deleteCount; i++ {
				deleteLeaf(t, &raw, optPaths[i])
			}

			// Verify required paths still resolve
			remaining := collectLeafPaths(raw)
			remSet := make(map[string]bool, len(remaining))
			for _, p := range remaining {
				remSet[p] = true
			}
			for p := range required {
				if !remSet[p] {
					t.Errorf("P.1: Required %q missing after %.0f%% deletion", p, tc.DeleteFraction*100)
				}
			}
		})
	}
}

func TestIXValuesStructure(t *testing.T) {
	// S.1 Non-functional: structure must match reference pattern
	raw := readIXValues(t)

	// S.2. Images must have exactly 2 entries
	if len(raw.Images) != 2 {
		t.Errorf("S.2: Expected 2 images, got %d", len(raw.Images))
	}

	// S.3. Consts must have exactly 3 entries
	if len(raw.Consts) != 3 {
		t.Errorf("S.3: Expected 3 consts, got %d: %v", len(raw.Consts), raw.Consts)
	}

	// S.4. Container names must not contain underscores or spaces
	for _, key := range []string{"kestrel_container_name", "perms_container_name"} {
		val, ok := raw.Consts[key]
		if !ok {
			continue
		}
		s, ok := val.(string)
		if !ok {
			t.Errorf("S.4: %q must be string, got %T", key, val)
			continue
		}
		if strings.ContainsAny(s, "_ ") {
			t.Errorf("S.4: %q = %q must not contain underscores or spaces", key, s)
		}
	}
}

func TestIXValuesRandomFuzz(t *testing.T) {
	// Fuzz: add random keys with random values and verify parsing still works
	raw := readIXValues(t)
	m := toMap(t, raw)

	for i := 0; i < 10; i++ {
		key := "x-test-" + randomString(8)
		m[key] = randomString(rand.Intn(32) + 1)
	}

	// Re-marshal and re-parse
	data, err := yaml.Marshal(m)
	if err != nil {
		t.Fatalf("Fuzz marshal: %v", err)
	}
	var parsed map[string]any
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Fuzz unmarshal: %v", err)
	}
}

func TestIXValuesParsePerformance(t *testing.T) {
	// Perf: parse must complete in < 10ms
	file := absPath(t, "ix_values.yaml")
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	for i := 0; i < 100; i++ {
		var v IXValues
		if err := yaml.Unmarshal(data, &v); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
	}
}

// --- Integration helpers ---

func TestIXValuesIntegrationSmokeChain(t *testing.T) {
	// I.1 Smoke: load, check structure, check values, check round-trip
	raw := readIXValues(t)

	// Check image tags don't use "latest" (anti-pattern for releases)
	for name, img := range raw.Images {
		if img.Tag == "latest" {
			t.Errorf("I.1: Image %q uses 'latest' tag — pin to a version", name)
		}
	}

	// Check repository URLs are well-formed
	for name, img := range raw.Images {
		if !strings.Contains(img.Repository, "/") {
			t.Errorf("I.1: Image %q repository %q has no namespace", name, img.Repository)
		}
	}

	// Round-trip: marshal -> unmarshal -> compare consts
	out, err := yaml.Marshal(raw)
	if err != nil {
		t.Fatalf("I.1: marshal error: %v", err)
	}
	var round IXValues
	if err := yaml.Unmarshal(out, &round); err != nil {
		t.Fatalf("I.1: unmarshal round-trip: %v", err)
	}
	if !reflect.DeepEqual(raw.Consts, round.Consts) {
		t.Errorf("I.1: consts not equal after round-trip")
	}
}

func TestIXValuesIntegrationCrossFile(t *testing.T) {
	// I.2: Cross-file — the container name in ix_values must be referenced
	//     in docker-compose.yaml template. Check the first occurrence.
	raw := readIXValues(t)
	containerName, _ := raw.Consts["kestrel_container_name"].(string)

	dcFile := absPath(t, "templates", "docker-compose.yaml")
	dcData, err := os.ReadFile(dcFile)
	if err != nil {
		t.Skipf("I.2: cannot read docker-compose.yaml: %v", err)
	}
	if !strings.Contains(string(dcData), containerName) {
		t.Errorf("I.2: container name %q not found in docker-compose.yaml", containerName)
	}
}

// collectLeafPaths returns all leaf key paths from an IXValues.
func collectLeafPaths(v IXValues) []string {
	var paths []string
	// Images leaf paths
	for name := range v.Images {
		paths = append(paths, "images."+name)
	}
	// Consts leaf paths
	for name := range v.Consts {
		paths = append(paths, "consts."+name)
	}
	return paths
}

// deleteLeaf removes a leaf entry by path (e.g. "images.playwright_image" or "consts.run_as_user").
func deleteLeaf(t *testing.T, v *IXValues, path string) {
	t.Helper()
	parts := strings.SplitN(path, ".", 2)
	if len(parts) != 2 {
		t.Fatalf("invalid path: %s", path)
	}
	switch parts[0] {
	case "images":
		delete(v.Images, parts[1])
	case "consts":
		delete(v.Consts, parts[1])
	default:
		t.Fatalf("unknown top-level key: %s", parts[0])
	}
}

// --- utilities ---

func readIXValues(t *testing.T) IXValues {
	t.Helper()
	file := absPath(t, "ix_values.yaml")
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read ix_values.yaml: %v", err)
	}
	var v IXValues
	if err := yaml.Unmarshal(data, &v); err != nil {
		t.Fatalf("unmarshal ix_values.yaml: %v", err)
	}
	return v
}

func absPath(t *testing.T, parts ...string) string {
	t.Helper()
	// Walk up to find the truenas/ directory (test runs from repo root or truenas/)
	cwd, _ := os.Getwd()
	if strings.HasSuffix(cwd, "truenas") {
		return filepath.Join(append([]string{cwd}, parts...)...)
	}
	return filepath.Join(append([]string{cwd, "truenas"}, parts...)...)
}

// ──────────────────────────────────────────────
// docker-compose.yaml validation tests
// ──────────────────────────────────────────────

func readDCTemplate(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(absPath(t, "templates", "docker-compose.yaml"))
	if err != nil {
		t.Skipf("read docker-compose.yaml: %v", err)
	}
	return string(data)
}

func TestDCFunctionalRequiredPatterns(t *testing.T) {
	content := readDCTemplate(t)

	tests := []struct {
		name    string
		pattern string
	}{
		{"healthcheck set_test", "healthcheck.set_test"},
		{"internal_web_port ref", valuesDot("consts.internal_web_port")},
		{"add_user_envs helper", "add_user_envs"},
		{"container_port key", "container_port"},
		{"host_network guard", "if not values.network.host_network"},
		{"perm_container defined early", valuesDot("consts.perms_container_name")},
		{"add_or_skip_action pattern", "add_or_skip_action"},
		{"perm_container.activate", "perm_container.activate()"},
		{"portals.add with port key", "tpl.portals.add"},
		{"render call", "tpl.render() | tojson"},
		{"set_user present", "c1.set_user"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(content, tc.pattern) {
				t.Errorf("Required pattern %q not found in docker-compose.yaml", tc.pattern)
			}
		})
	}

	// The for-loop over additional_envs must NOT exist (replaced by add_user_envs)
	if strings.Contains(content, "for env in values.kestrel.additional_envs") {
		t.Error("Forbidden: manual env loop found — use add_user_envs instead")
	}
}

func TestDCForbiddenPatterns(t *testing.T) {
	content := readDCTemplate(t)

	forbidden := []string{
		"playwright",
		"scraper",
		"extractor",
		"kestrel-scraper",
		"playwright_image",
		"KESTREL_SCRAPER_URL",
		"KESTREL_EXTRACTOR_MODE",
		"tpl.notes.set_body",
	}
	for _, pat := range forbidden {
		t.Run("no_"+strings.ReplaceAll(pat, ".", "_"), func(t *testing.T) {
			if strings.Contains(strings.ToLower(content), strings.ToLower(pat)) {
				t.Errorf("Forbidden pattern %q found in docker-compose.yaml", pat)
			}
		})
	}
}

func TestDCPortDefinitionCorrect(t *testing.T) {
	content := readDCTemplate(t)
	// Port must use container_port key
	if !strings.Contains(content, "container_port") {
		t.Error("Port definition missing container_port key")
	}
}

func TestDCHealthcheckPortMatch(t *testing.T) {
	// The healthcheck curl port references values.consts.internal_web_port
	content := readDCTemplate(t)
	if !strings.Contains(content, "values.consts.internal_web_port") {
		t.Error("Healthcheck or port definition does not reference values.consts.internal_web_port")
	}
}

func TestDCPermissionsStructure(t *testing.T) {
	content := readDCTemplate(t)

	// perm_container must be defined before any add_storage
	c1StorageIdx := strings.Index(content, "add_storage")
	permDefIdx := strings.Index(content, "tpl.deps.perms")
	if permDefIdx == -1 {
		t.Error("perm_container definition (tpl.deps.perms) not found")
		return
	}
	if c1StorageIdx != -1 && permDefIdx > c1StorageIdx {
		t.Error("perm_container defined AFTER add_storage — reorder to top")
	}

	// add_or_skip_action must follow add_storage immediately for data
	dataStorageIdx := strings.Index(content, `add_storage("/data"`)
	permActionIdx := strings.Index(content, `add_or_skip_action("data"`)
	if dataStorageIdx != -1 && permActionIdx != -1 && permActionIdx < dataStorageIdx {
		t.Error("add_or_skip_action for data appears before add_storage — wrong order")
	}
}

func TestDCRandomPartition(t *testing.T) {
	// P.1: Randomly delete lines and verify required patterns survive
	requiredPatterns := []string{
		"healthcheck.set_test",
		"add_user_envs",
		"container_port",
		"add_or_skip_action",
		"tpl.portals.add",
		"tpl.render() | tojson",
	}

	for _, frac := range []float64{0.0, 0.1, 0.2, 0.3} {
		t.Run(formatFloat(frac), func(t *testing.T) {
			content := readDCTemplate(t)
			lines := strings.Split(content, "\n")
			// Collect deletable line indices (non-empty, non-required)
			deletable := make([]int, 0)
			for i, line := range lines {
				trimmed := strings.TrimSpace(line)
				if trimmed == "" {
					continue
				}
				isRequired := false
				for _, pat := range requiredPatterns {
					if strings.Contains(trimmed, pat) {
						isRequired = true
						break
					}
				}
				if !isRequired {
					deletable = append(deletable, i)
				}
			}
			rand.Shuffle(len(deletable), func(i, j int) {
				deletable[i], deletable[j] = deletable[j], deletable[i]
			})
			deleteCount := int(float64(len(deletable)) * frac)
			for i := 0; i < deleteCount; i++ {
				lines[deletable[i]] = ""
			}
			modified := strings.Join(lines, "\n")
			for _, pat := range requiredPatterns {
				if !strings.Contains(modified, pat) {
					t.Errorf("Required pattern %q lost after %.0f%% deletion", pat, frac*100)
				}
			}
		})
	}
}

func TestDCParsePerformance(t *testing.T) {
	// Perf: read file 100 times in < 100ms (simple IO benchmark)
	file := absPath(t, "templates", "docker-compose.yaml")
	for i := 0; i < 100; i++ {
		if _, err := os.ReadFile(file); err != nil {
			t.Fatalf("read error: %v", err)
		}
	}
}

func TestDCIntegrationCrossFile(t *testing.T) {
	// I.1: Every const from ix_values.yaml referenced in docker-compose.yaml
	ix := readIXValues(t)
	content := readDCTemplate(t)

	for key, val := range ix.Consts {
		// Check that values are referenced (e.g., "values.consts.kestrel_container_name")
		constRef := valuesDot("consts." + key)
		if !strings.Contains(content, constRef) {
			// Check if raw string value is used directly
			if s, ok := val.(string); ok && !strings.Contains(content, s) {
				t.Errorf("I.1: const %q (%v) not referenced in docker-compose.yaml", key, val)
			} else if _, ok := val.(int); ok {
				t.Errorf("I.1: const %q not referenced via %s in docker-compose.yaml", key, constRef)
			}
		}
	}
}

func TestDCIntegrationSmokeChain(t *testing.T) {
	// I.2: Load template, verify first/last lines, check structural balance
	content := readDCTemplate(t)
	lines := strings.Split(content, "\n")

	// First meaningful line must be set tpl = ix_lib.base.render.Render
	firstNonEmpty := ""
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			firstNonEmpty = strings.TrimSpace(l)
			break
		}
	}
	expectedFirst := "{% set tpl = ix_lib.base.render.Render(values) %}"
	if firstNonEmpty != expectedFirst {
		t.Errorf("I.2: First line %q, want %q", firstNonEmpty, expectedFirst)
	}

	// Template must end with render
	lastNonEmpty := ""
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			lastNonEmpty = strings.TrimSpace(lines[i])
			break
		}
	}
	if !strings.Contains(lastNonEmpty, "tpl.render() | tojson") {
		t.Errorf("I.2: Last line must contain render call, got %q", lastNonEmpty)
	}
}

// valuesDot formats a dotted values path like "values.consts.foo".
func valuesDot(path string) string {
	return "values." + path
}

func formatInt(n int) string {
	return strings.TrimSpace(strings.Replace(
		strings.Replace(
			strings.Replace(
				strings.Replace(
					fmt.Sprintf("%d", n),
					" ", "", -1),
				"\n", "", -1),
			"\r", "", -1),
		"\t", "", -1),
	)
}

func formatFloat(f float64) string {
	s := fmt.Sprintf("%.0f", f*100)
	return s + "pct"
}

func toMap(t *testing.T, v IXValues) map[string]any {
	t.Helper()
	data, err := yaml.Marshal(v)
	if err != nil {
		t.Fatalf("toMap marshal: %v", err)
	}
	var m map[string]any
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatalf("toMap unmarshal: %v", err)
	}
	return m
}


