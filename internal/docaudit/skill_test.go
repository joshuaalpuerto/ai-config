package docaudit_test

import (
	"strings"
	"testing"

	"github.com/joshuaalpuerto/ai-config/internal/docaudit"
)

func TestGenerateSkill_containsRequiredFrontmatter(t *testing.T) {
	skill := docaudit.GenerateSkill(docaudit.Config{
		TargetDir:   ".",
		ProjectName: "testproject",
		DocRoots:    []string{"docs/"},
	})

	for _, want := range []string{"name: doc-audit", "description:", "allowed-tools:", "- Agent"} {
		if !strings.Contains(skill, want) {
			t.Errorf("expected skill frontmatter to contain %q", want)
		}
	}
}

func TestGenerateSkill_embedsConfig(t *testing.T) {
	cfg := docaudit.Config{
		TargetDir:      "/home/user/myproject",
		ProjectName:    "myproject",
		DocRoots:       []string{"docs/", "README.md"},
		AnalyzeExclude: []string{"vendor/", "dist/"},
	}
	skill := docaudit.GenerateSkill(cfg)

	if !strings.Contains(skill, cfg.TargetDir) {
		t.Errorf("expected skill to contain target dir %q", cfg.TargetDir)
	}
	for _, root := range cfg.DocRoots {
		if !strings.Contains(skill, root) {
			t.Errorf("expected skill to contain doc root %q", root)
		}
	}
	for _, excl := range cfg.AnalyzeExclude {
		if !strings.Contains(skill, excl) {
			t.Errorf("expected skill to contain exclude pattern %q", excl)
		}
	}
}

func TestGenerateSkill_containsCoreSections(t *testing.T) {
	skill := docaudit.GenerateSkill(docaudit.Config{
		TargetDir:   ".",
		ProjectName: "testproject",
		DocRoots:    []string{"docs/"},
	})

	requiredSections := []string{
		"## Project Configuration",
		"## Process",
		"## Output Format",
		"### Contributor Blockers",
		"### Undocumented Contracts",
		"### Complexity Traps",
		"### Docs Needing Updates",
		"### Undocumented Dependency Conventions",
		"### Suggested Actions",
		"#### Task A",
		"#### Task B",
		"#### Task C",
		"aicfg analyze --kind=doc",
	}
	for _, section := range requiredSections {
		if !strings.Contains(skill, section) {
			t.Errorf("expected skill to contain section %q", section)
		}
	}
}

func TestGenerateSkill_noDocRoots_formatsAsNone(t *testing.T) {
	skill := docaudit.GenerateSkill(docaudit.Config{
		TargetDir:   ".",
		ProjectName: "testproject",
		DocRoots:    []string{},
	})

	if !strings.Contains(skill, "none") {
		t.Error("expected skill to show 'none' when no doc roots provided")
	}
}

func TestGenerateSkill_noAnalyzeExclude_omitsExcludeField(t *testing.T) {
	skill := docaudit.GenerateSkill(docaudit.Config{
		TargetDir:      ".",
		ProjectName:    "testproject",
		DocRoots:       []string{"docs/"},
		AnalyzeExclude: nil,
	})

	if strings.Contains(skill, "Analyze excludes") {
		t.Error("expected skill to omit 'Analyze excludes' when no exclusions configured")
	}
}

func TestGenerateSkill_descriptionContainsProjectName(t *testing.T) {
	skill := docaudit.GenerateSkill(docaudit.Config{
		TargetDir:   ".",
		ProjectName: "my-cool-app",
		DocRoots:    []string{"docs/"},
	})

	if !strings.Contains(skill, "my-cool-app") {
		t.Error("expected skill description to contain the project name")
	}
}
