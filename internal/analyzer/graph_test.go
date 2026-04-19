package analyzer

import (
	"testing"
)

func TestComputePriority_EntryPoint(t *testing.T) {
	nodes := map[string]*graphNode{
		"cmd/main.go": {
			path:         "cmd/main.go",
			isEntryPoint: true,
			folderDepth:  1,
		},
	}
	computePriorities(nodes)

	node := nodes["cmd/main.go"]
	if node.priority < 10.0 {
		t.Errorf("entry point should have priority >= 10, got %f", node.priority)
	}
}

func TestComputePriority_HubFile(t *testing.T) {
	nodes := map[string]*graphNode{
		"internal/utils/utils.go": {
			path:         "internal/utils/utils.go",
			importedBy:   []string{"a.go", "b.go", "c.go"},
			exportCount:  5,
			folderDepth:  2,
			churn:        4,
		},
	}
	computePriorities(nodes)

	node := nodes["internal/utils/utils.go"]
	// fanIn=3*3 + exports=5*1.5 + churn=4*2 = 9+7.5+8 = 24.5 + depth bonus ~0.33
	expected := (3.0 * 3.0) + (5.0 * 1.5) + (4.0 * 2.0) + (1.0 / 3.0)
	if node.priority < expected-0.1 || node.priority > expected+0.1 {
		t.Errorf("expected priority ~%.2f, got %.2f", expected, node.priority)
	}
}

func TestTopHubs_OrderedByPriority(t *testing.T) {
	nodes := map[string]*graphNode{
		"low.go":  {path: "low.go", priority: 1.0},
		"high.go": {path: "high.go", priority: 50.0},
		"mid.go":  {path: "mid.go", priority: 20.0},
	}
	hubs := topHubs(nodes, 10)
	if len(hubs) != 3 {
		t.Fatalf("expected 3 hubs, got %d", len(hubs))
	}
	if hubs[0].Path != "high.go" {
		t.Errorf("expected high.go first, got %s", hubs[0].Path)
	}
	if hubs[1].Path != "mid.go" {
		t.Errorf("expected mid.go second, got %s", hubs[1].Path)
	}
}

func TestTopHubs_LimitN(t *testing.T) {
	nodes := make(map[string]*graphNode)
	for i := 0; i < 20; i++ {
		key := "file" + string(rune('a'+i)) + ".go"
		nodes[key] = &graphNode{path: key, priority: float64(i)}
	}
	hubs := topHubs(nodes, 5)
	if len(hubs) != 5 {
		t.Errorf("expected 5 hubs (limit), got %d", len(hubs))
	}
}

func TestIsEntryPoint(t *testing.T) {
	cases := []struct {
		path     string
		expected bool
	}{
		{"cmd/aicfg/main.go", true},
		{"main.go", true},
		{"internal/utils/utils.go", false},
		{"src/index.ts", true},
		{"src/app.tsx", true},
		{"src/server.js", true},
		{"src/components/Button.tsx", false},
		{"__main__.py", true},
		{"main.py", true},
		{"app/views.py", false},
	}
	for _, c := range cases {
		got := isEntryPoint(c.path)
		if got != c.expected {
			t.Errorf("isEntryPoint(%q) = %v, want %v", c.path, got, c.expected)
		}
	}
}
