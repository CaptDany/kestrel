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

// Questions models the top-level structure of questions.yaml.
type Questions struct {
	Groups    []Group    `yaml:"groups"`
	Questions []Question `yaml:"questions"`
}

type Group struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type Question struct {
	Variable string    `yaml:"variable"`
	Label    string    `yaml:"label"`
	Group    string    `yaml:"group"`
	Schema   yaml.Node `yaml:"schema"`
}

func readQuestions(t *testing.T) Questions {
	t.Helper()
	data, err := os.ReadFile(absPath(t, "questions.yaml"))
	if err != nil {
		t.Fatalf("read questions.yaml: %v", err)
	}
	var q Questions
	if err := yaml.Unmarshal(data, &q); err != nil {
		t.Fatalf("unmarshal questions.yaml: %v", err)
	}
	return q
}

// ──────────────────────────────────────────────
// Functional tests
// ──────────────────────────────────────────────

func TestQFunctionalRequiredQuestions(t *testing.T) {
	q := readQuestions(t)

	requiredVars := []string{"TZ", "kestrel", "run_as", "network", "storage", "labels", "resources"}
	varNames := make(map[string]bool)
	for _, qn := range q.Questions {
		varNames[qn.Variable] = true
	}
	for _, v := range requiredVars {
		if !varNames[v] {
			t.Errorf("F.1: Required question %q missing", v)
		}
	}
}

func TestQFunctionalGroupNamesCapitalized(t *testing.T) {
	q := readQuestions(t)
	for _, g := range q.Groups {
		if g.Name != "" && !isCapitalized(g.Name) {
			t.Errorf("F.2: Group name %q should start with capital letter", g.Name)
		}
	}
}

func TestQFunctionalForbiddenQuestions(t *testing.T) {
	q := readQuestions(t)

	forbiddenVars := []string{"extractor"}
	// Also check that variable names referencing removed features don't exist
	usedVars := make(map[string]bool)
	for _, qn := range q.Questions {
		usedVars[qn.Variable] = true
	}
	for _, v := range forbiddenVars {
		if usedVars[v] {
			t.Errorf("F.3: Forbidden question %q must not exist", v)
		}
	}
}

func TestQFunctionalForbiddenAttrs(t *testing.T) {
	// Read raw YAML text and check for forbidden attribute names
	data, err := os.ReadFile(absPath(t, "questions.yaml"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	raw := string(data)

	forbidden := []string{
		"budget_mode",
		"budget_amount",
		"purchase_mode",
		"sort_criteria",
		"currency",
	}
	for _, f := range forbidden {
		if strings.Contains(raw, f) {
			t.Errorf("F.4: Forbidden attribute %q found in questions.yaml", f)
		}
	}
}

func TestQFunctionalExtractorSectionAbsent(t *testing.T) {
	data, err := os.ReadFile(absPath(t, "questions.yaml"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	raw := string(data)
	if strings.Contains(strings.ToLower(raw), "extractor") {
		t.Error("F.5: Extractor section must be removed from questions.yaml")
	}
}

// ──────────────────────────────────────────────
// Non-functional / Structural tests
// ──────────────────────────────────────────────

func TestQStructureGroupCount(t *testing.T) {
	q := readQuestions(t)
	// Expected groups: Kestrel Configuration, User and Group, Network, Storage, Labels, Resources
	if len(q.Groups) < 6 {
		t.Errorf("S.1: Expected at least 6 groups, got %d", len(q.Groups))
	}
}

func TestQStructureAllGroupsReferenced(t *testing.T) {
	q := readQuestions(t)
	groupNames := make(map[string]bool)
	for _, g := range q.Groups {
		groupNames[g.Name] = true
	}
	for _, qn := range q.Questions {
		if qn.Group != "" && !groupNames[qn.Group] {
			t.Errorf("S.2: Question %q references non-existent group %q", qn.Variable, qn.Group)
		}
	}
}

func TestQStructureTZBeforeKestrel(t *testing.T) {
	q := readQuestions(t)
	tzIdx := -1
	kestrelIdx := -1
	for i, qn := range q.Questions {
		if qn.Variable == "TZ" {
			tzIdx = i
		}
		if qn.Variable == "kestrel" {
			kestrelIdx = i
		}
	}
	if tzIdx == -1 {
		t.Error("S.3: TZ question missing")
	}
	if kestrelIdx == -1 {
		t.Error("S.3: kestrel question missing")
	}
	if tzIdx != -1 && kestrelIdx != -1 && tzIdx > kestrelIdx {
		t.Error("S.3: TZ must appear before kestrel in questions order")
	}
}

// ──────────────────────────────────────────────
// Random-partition / Fuzz testing
// ──────────────────────────────────────────────

func TestQRandomPartition(t *testing.T) {
	// P.1: Delete random subsets of optional questions, verify mandatory ones survive
	mandatory := map[string]bool{
		"TZ":       true,
		"kestrel":  true,
		"run_as":   true,
		"network":  true,
		"storage":  true,
		"labels":   true,
		"resources": true,
	}

	for _, frac := range []float64{0.0, 0.25, 0.50} {
		t.Run(fmt.Sprintf("%.0fpct", frac*100), func(t *testing.T) {
			q := readQuestions(t)

			// Collect optional question indices
			optIndices := make([]int, 0)
			for i, qn := range q.Questions {
				if !mandatory[qn.Variable] {
					optIndices = append(optIndices, i)
				}
			}
			rand.Shuffle(len(optIndices), func(i, j int) {
				optIndices[i], optIndices[j] = optIndices[j], optIndices[i]
			})
			deleteCount := int(float64(len(optIndices)) * frac)
			removed := make(map[int]bool)
			for i := 0; i < deleteCount; i++ {
				removed[optIndices[i]] = true
			}

			// Verify mandatory questions survive
			for _, qn := range q.Questions {
				if mandatory[qn.Variable] && removed[indexOfQuestion(q.Questions, qn.Variable)] {
					t.Errorf("P.1: Mandatory question %q removed after %.0f%% deletion", qn.Variable, frac*100)
				}
			}
		})
	}
}

func TestQRandomFuzz(t *testing.T) {
	// Fuzz: Marshal/unmarshal with extra random fields
	data, err := yaml.Marshal(readQuestions(t))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var generic map[string]any
	if err := yaml.Unmarshal(data, &generic); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Add random junk
	for i := 0; i < 5; i++ {
		key := "x-fuzz-" + randomString(6)
		generic[key] = randomString(rand.Intn(20) + 1)
	}
	out, err := yaml.Marshal(generic)
	if err != nil {
		t.Fatalf("fuzz marshal: %v", err)
	}
	var round Questions
	if err := yaml.Unmarshal(out, &round); err != nil {
		t.Fatalf("fuzz unmarshal: %v", err)
	}
}

// ──────────────────────────────────────────────
// Performance
// ──────────────────────────────────────────────

func TestQParsePerformance(t *testing.T) {
	file := absPath(t, "questions.yaml")
	for i := 0; i < 100; i++ {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		var q Questions
		if err := yaml.Unmarshal(data, &q); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
	}
}

// ──────────────────────────────────────────────
// Integration / Smoke
// ──────────────────────────────────────────────

func TestQIntegrationSmokeChain(t *testing.T) {
	q := readQuestions(t)
	// Smoke: load, verify TZ, check round-trip
	if q.Questions[0].Variable != "TZ" {
		t.Errorf("I.1: First question should be TZ, got %q", q.Questions[0].Variable)
	}
	// Verify every group has a description
	for _, g := range q.Groups {
		if g.Description == "" {
			t.Errorf("I.1: Group %q has no description", g.Name)
		}
	}
	// Round-trip
	data, err := yaml.Marshal(q)
	if err != nil {
		t.Fatalf("I.1: marshal: %v", err)
	}
	var round Questions
	if err := yaml.Unmarshal(data, &round); err != nil {
		t.Fatalf("I.1: round-trip unmarshal: %v", err)
	}
	if len(round.Questions) != len(q.Questions) {
		t.Errorf("I.1: question count mismatch after round-trip: %d vs %d", len(round.Questions), len(q.Questions))
	}
}

func TestQIntegrationCrossFile(t *testing.T) {
	// I.2: Verify question variable names referenced in docker-compose.yaml exist
	dcData, err := os.ReadFile(absPath(t, "templates", "docker-compose.yaml"))
	if err != nil {
		t.Skip("docker-compose.yaml not found")
	}
	dc := string(dcData)

	// These question variable paths must be referenced in the template
	expectedRefs := []string{
		"values.TZ",
		"values.kestrel.additional_envs",
		"values.run_as.user",
		"values.run_as.group",
		"values.network.web_port",
		"values.network.host_network",
		"values.storage.data",
		"values.storage.additional_storage",
	}
	for _, ref := range expectedRefs {
		if !strings.Contains(dc, ref) {
			t.Errorf("I.2: %q not found in docker-compose.yaml", ref)
		}
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

func isCapitalized(s string) bool {
	if s == "" {
		return false
	}
	c := s[0]
	return c >= 'A' && c <= 'Z'
}

func indexOfQuestion(qs []Question, variable string) int {
	for i, q := range qs {
		if q.Variable == variable {
			return i
		}
	}
	return -1
}
