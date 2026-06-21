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

// ──────────────────────────────────────────────
// Models
// ──────────────────────────────────────────────

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

type IXValues struct {
	Images map[string]ImageInfo `yaml:"images"`
	Consts map[string]any       `yaml:"consts"`
}

type ImageInfo struct {
	Repository string `yaml:"repository"`
	Tag        string `yaml:"tag"`
}

// ──────────────────────────────────────────────
// Shared utilities
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
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func toMapIX(v IXValues) (map[string]any, error) {
	data, err := yaml.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// ──────────────────────────────────────────────
// AppYAML helpers & tests
// ──────────────────────────────────────────────

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

func TestAppFunctionalLibVersionHash(t *testing.T) {
	a := readAppYAML(t)
	if a.LibVersionHash == "" {
		t.Error("F.1: lib_version_hash must not be empty")
	}
	if len(a.LibVersionHash) != 64 {
		t.Errorf("F.1: lib_version_hash length %d, expected 64", len(a.LibVersionHash))
	}
	for _, c := range a.LibVersionHash {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			t.Errorf("F.1: lib_version_hash contains invalid hex char %c", c)
			break
		}
	}
}

func TestAppFunctionalRunAsContextCount(t *testing.T) {
	a := readAppYAML(t)
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

func TestAppRandomPartition(t *testing.T) {
	critical := map[string]bool{
		"lib_version":      true,
		"app_version":      true,
		"lib_version_hash": true,
		"name":             true,
		"train":            true,
		"version":          true,
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
// IXValues helpers & tests
// ──────────────────────────────────────────────

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

func TestIXValuesFunctionalRequiredKeysExist(t *testing.T) {
	raw := readIXValues(t)

	requiredImages := []string{"image", "container_utils_image"}
	for _, img := range requiredImages {
		if _, ok := raw.Images[img]; !ok {
			t.Errorf("Required image %q missing", img)
		}
	}

	requiredConsts := []string{"kestrel_container_name", "perms_container_name", "internal_web_port"}
	for _, c := range requiredConsts {
		if _, ok := raw.Consts[c]; !ok {
			t.Errorf("Required const %q missing", c)
		}
	}

	for name, img := range raw.Images {
		if img.Repository == "" {
			t.Errorf("Image %q has empty repository", name)
		}
		if img.Tag == "" {
			t.Errorf("Image %q has empty tag", name)
		}
	}

	switch v := raw.Consts["internal_web_port"].(type) {
	case int:
		if v <= 0 || v > 65535 {
			t.Errorf("internal_web_port out of range: %d", v)
		}
	default:
		t.Errorf("internal_web_port must be int, got %T", v)
	}
}

func TestIXValuesForbiddenKeysAbsent(t *testing.T) {
	raw := readIXValues(t)

	forbiddenImages := []string{"playwright_image"}
	for _, img := range forbiddenImages {
		if _, ok := raw.Images[img]; ok {
			t.Errorf("Forbidden image %q must not exist", img)
		}
	}

	forbiddenConsts := []string{"scraper_container_name", "run_as_user", "run_as_group"}
	for _, c := range forbiddenConsts {
		if _, ok := raw.Consts[c]; ok {
			t.Errorf("Forbidden const %q must not exist", c)
		}
	}
}

func TestIXValuesRandomPartition(t *testing.T) {
	required := map[string]bool{
		"images.image":                  true,
		"images.container_utils_image":  true,
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

			allPaths := collectLeafPaths(raw)
			optPaths := make([]string, 0, len(allPaths))
			for _, p := range allPaths {
				if !required[p] {
					optPaths = append(optPaths, p)
				}
			}

			rand.Shuffle(len(optPaths), func(i, j int) {
				optPaths[i], optPaths[j] = optPaths[j], optPaths[i]
			})
			deleteCount := int(float64(len(optPaths)) * tc.DeleteFraction)
			for i := 0; i < deleteCount; i++ {
				deleteLeaf(t, &raw, optPaths[i])
			}

			remaining := collectLeafPaths(raw)
			remSet := make(map[string]bool, len(remaining))
			for _, p := range remaining {
				remSet[p] = true
			}
			for p := range required {
				if !remSet[p] {
					t.Errorf("Required %q missing after %.0f%% deletion", p, tc.DeleteFraction*100)
				}
			}
		})
	}
}

func TestIXValuesStructure(t *testing.T) {
	raw := readIXValues(t)

	if len(raw.Images) != 2 {
		t.Errorf("Expected 2 images, got %d", len(raw.Images))
	}
	if len(raw.Consts) != 3 {
		t.Errorf("Expected 3 consts, got %d", len(raw.Consts))
	}

	for _, key := range []string{"kestrel_container_name", "perms_container_name"} {
		val, ok := raw.Consts[key]
		if !ok {
			continue
		}
		s, ok := val.(string)
		if !ok {
			t.Errorf("%q must be string, got %T", key, val)
			continue
		}
		if strings.ContainsAny(s, "_ ") {
			t.Errorf("%q = %q must not contain underscores or spaces", key, s)
		}
	}
}

func TestIXValuesRandomFuzz(t *testing.T) {
	raw := readIXValues(t)
	m, err := toMapIX(raw)
	if err != nil {
		t.Fatalf("toMap: %v", err)
	}

	for i := 0; i < 10; i++ {
		key := "x-test-" + randomString(8)
		m[key] = randomString(rand.Intn(32) + 1)
	}

	data, err := yaml.Marshal(m)
	if err != nil {
		t.Fatalf("fuzz marshal: %v", err)
	}
	var parsed map[string]any
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("fuzz unmarshal: %v", err)
	}
}

func TestIXValuesParsePerformance(t *testing.T) {
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

func TestIXValuesIntegrationSmokeChain(t *testing.T) {
	raw := readIXValues(t)

	for name, img := range raw.Images {
		if img.Tag == "latest" {
			t.Errorf("Image %q uses 'latest' tag", name)
		}
		if !strings.Contains(img.Repository, "/") {
			t.Errorf("Image %q repository %q has no namespace", name, img.Repository)
		}
	}

	out, err := yaml.Marshal(raw)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var round IXValues
	if err := yaml.Unmarshal(out, &round); err != nil {
		t.Fatalf("round-trip unmarshal: %v", err)
	}
	if !reflect.DeepEqual(raw.Consts, round.Consts) {
		t.Errorf("consts not equal after round-trip")
	}
}

func TestIXValuesIntegrationCrossFile(t *testing.T) {
	raw := readIXValues(t)
	containerName, _ := raw.Consts["kestrel_container_name"].(string)

	dcFile := absPath(t, "templates", "docker-compose.yaml")
	dcData, err := os.ReadFile(dcFile)
	if err != nil {
		t.Skipf("cannot read docker-compose.yaml: %v", err)
	}
	if !strings.Contains(string(dcData), containerName) {
		t.Errorf("container name %q not found in docker-compose.yaml", containerName)
	}
}

// collectLeafPaths returns all leaf key paths from an IXValues.
func collectLeafPaths(v IXValues) []string {
	var paths []string
	for name := range v.Images {
		paths = append(paths, "images."+name)
	}
	for name := range v.Consts {
		paths = append(paths, "consts."+name)
	}
	return paths
}

// deleteLeaf removes a leaf entry by path (e.g. "images.playwright_image").
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

// ──────────────────────────────────────────────
// Docker-compose template tests
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
	if !strings.Contains(content, "container_port") {
		t.Error("Port definition missing container_port key")
	}
}

func TestDCHealthcheckPortMatch(t *testing.T) {
	content := readDCTemplate(t)
	if !strings.Contains(content, "values.consts.internal_web_port") {
		t.Error("Healthcheck does not reference values.consts.internal_web_port")
	}
}

func TestDCPermissionsStructure(t *testing.T) {
	content := readDCTemplate(t)

	c1StorageIdx := strings.Index(content, "add_storage")
	permDefIdx := strings.Index(content, "tpl.deps.perms")
	if permDefIdx == -1 {
		t.Error("perm_container definition (tpl.deps.perms) not found")
		return
	}
	if c1StorageIdx != -1 && permDefIdx > c1StorageIdx {
		t.Error("perm_container defined AFTER add_storage — reorder to top")
	}

	dataStorageIdx := strings.Index(content, `add_storage("/data"`)
	permActionIdx := strings.Index(content, `add_or_skip_action("data"`)
	if dataStorageIdx != -1 && permActionIdx != -1 && permActionIdx < dataStorageIdx {
		t.Error("add_or_skip_action for data appears before add_storage — wrong order")
	}
}

func TestDCRandomPartition(t *testing.T) {
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
	file := absPath(t, "templates", "docker-compose.yaml")
	for i := 0; i < 100; i++ {
		if _, err := os.ReadFile(file); err != nil {
			t.Fatalf("read error: %v", err)
		}
	}
}

func TestDCIntegrationCrossFile(t *testing.T) {
	ix := readIXValues(t)
	content := readDCTemplate(t)

	for key, val := range ix.Consts {
		constRef := valuesDot("consts." + key)
		if !strings.Contains(content, constRef) {
			if s, ok := val.(string); ok && !strings.Contains(content, s) {
				t.Errorf("const %q (%v) not referenced in docker-compose.yaml", key, val)
			} else if _, ok := val.(int); ok {
				t.Errorf("const %q not referenced via %s in docker-compose.yaml", key, constRef)
			}
		}
	}
}

func TestDCIntegrationSmokeChain(t *testing.T) {
	content := readDCTemplate(t)
	lines := strings.Split(content, "\n")

	firstNonEmpty := ""
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			firstNonEmpty = strings.TrimSpace(l)
			break
		}
	}
	expectedFirst := "{% set tpl = ix_lib.base.render.Render(values) %}"
	if firstNonEmpty != expectedFirst {
		t.Errorf("First line %q, want %q", firstNonEmpty, expectedFirst)
	}

	lastNonEmpty := ""
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			lastNonEmpty = strings.TrimSpace(lines[i])
			break
		}
	}
	if !strings.Contains(lastNonEmpty, "tpl.render() | tojson") {
		t.Errorf("Last line must contain render call, got %q", lastNonEmpty)
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

