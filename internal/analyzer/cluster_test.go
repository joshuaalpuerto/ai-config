package analyzer

import (
	"testing"
)

func TestClusterKey_TwoSegments(t *testing.T) {
	cases := []struct {
		path   string
		expect string
	}{
		{"internal/hooks/engine.go", "internal/hooks"},
		{"internal/analyzer/parser/go.go", "internal/analyzer"},
		{"cmd/aicfg/main.go", "cmd/aicfg"},
		{"schemas/schemas.go", "schemas"},
		{"main.go", "main"},
	}
	for _, c := range cases {
		got := clusterKey(c.path, "")
		if got != c.expect {
			t.Errorf("clusterKey(%q, \"\") = %q, want %q", c.path, got, c.expect)
		}
	}
}

func TestClusterKey_StripsSourceRoot(t *testing.T) {
	cases := []struct {
		path           string
		sourceRootName string
		expect         string
	}{
		{"src/components/Button.tsx", "src", "components"},
		{"src/hooks/useAuth.ts", "", "hooks"},
		{"lib/utils/string.py", "", "utils"},
	}
	for _, c := range cases {
		got := clusterKey(c.path, c.sourceRootName)
		if got != c.expect {
			t.Errorf("clusterKey(%q, %q) = %q, want %q", c.path, c.sourceRootName, got, c.expect)
		}
	}
}

func TestBuildClusters_Deterministic(t *testing.T) {
	nodes := map[string]*graphNode{
		"internal/hooks/engine.go": {path: "internal/hooks/engine.go"},
		"internal/hooks/models.go": {path: "internal/hooks/models.go"},
		"internal/config/types.go": {path: "internal/config/types.go"},
		"cmd/aicfg/main.go":        {path: "cmd/aicfg/main.go"},
	}

	c1 := buildClusters(nodes, "")
	c2 := buildClusters(nodes, "")

	if len(c1) != len(c2) {
		t.Fatalf("cluster count not deterministic: %d vs %d", len(c1), len(c2))
	}
	for i := range c1 {
		if c1[i].Label != c2[i].Label {
			t.Errorf("cluster[%d] label not deterministic: %q vs %q", i, c1[i].Label, c2[i].Label)
		}
	}
}

func TestBuildClusters_GroupsByDirectory(t *testing.T) {
	nodes := map[string]*graphNode{
		"internal/hooks/engine.go": {path: "internal/hooks/engine.go"},
		"internal/hooks/models.go": {path: "internal/hooks/models.go"},
		"internal/hooks/loader.go": {path: "internal/hooks/loader.go"},
		"internal/config/types.go": {path: "internal/config/types.go"},
	}

	clusters := buildClusters(nodes, "")

	// Should have 2 clusters: internal/hooks (3 files) and internal/config (1 file).
	if len(clusters) != 2 {
		t.Fatalf("expected 2 clusters, got %d", len(clusters))
	}

	// Sorted by size desc — hooks first.
	if clusters[0].Label != "internal/hooks" {
		t.Errorf("expected first cluster 'internal/hooks', got %q", clusters[0].Label)
	}
	if clusters[0].Size != 3 {
		t.Errorf("expected hooks cluster size 3, got %d", clusters[0].Size)
	}
}

func TestBuildClusters_MergesSubpackages(t *testing.T) {
	// Parser files under internal/analyzer/parser/ should merge into internal/analyzer.
	nodes := map[string]*graphNode{
		"internal/analyzer/analyzer.go":  {path: "internal/analyzer/analyzer.go"},
		"internal/analyzer/parser/go.go": {path: "internal/analyzer/parser/go.go"},
		"internal/analyzer/parser/js.go": {path: "internal/analyzer/parser/js.go"},
	}

	clusters := buildClusters(nodes, "")

	if len(clusters) != 1 {
		t.Fatalf("expected 1 cluster (merged sub-packages), got %d", len(clusters))
	}
	if clusters[0].Label != "internal/analyzer" {
		t.Errorf("expected label 'internal/analyzer', got %q", clusters[0].Label)
	}
}

func TestBuildClusters_InterClusterDeps(t *testing.T) {
	nodes := map[string]*graphNode{
		"cmd/aicfg/main.go":        {path: "cmd/aicfg/main.go", imports: []string{"internal/hooks/engine.go"}},
		"internal/hooks/engine.go": {path: "internal/hooks/engine.go", importedBy: []string{"cmd/aicfg/main.go"}},
	}

	clusters := buildClusters(nodes, "")

	var cmdCluster *Cluster
	for i := range clusters {
		if clusters[i].Label == "cmd/aicfg" {
			cmdCluster = &clusters[i]
			break
		}
	}
	if cmdCluster == nil {
		t.Fatal("expected cmd/aicfg cluster")
	}
	if len(cmdCluster.DependsOn) == 0 {
		t.Error("expected cmd/aicfg to depend on internal/hooks")
	}
}
