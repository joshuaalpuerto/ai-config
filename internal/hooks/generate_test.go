package hooks_test

import (
	"strings"
	"testing"

	"github.com/joshuaalpuerto/ai-config/internal/hooks"
)

func TestGenerateSkill_containsRequiredFrontmatter(t *testing.T) {
	skill := hooks.GenerateSkill(hooks.GenerateConfig{
		TargetDir:   ".",
		ProjectName: "testproject",
		DocRoots:    []string{"docs/"},
	})

	for _, want := range []string{"name: hooks-generator", "description:", "allowed-tools:", "- Write"} {
		if !strings.Contains(skill, want) {
			t.Errorf("expected skill frontmatter to contain %q", want)
		}
	}
}

func TestGenerateSkill_embedsConfig(t *testing.T) {
	cfg := hooks.GenerateConfig{
		TargetDir:      "/home/user/myproject",
		ProjectName:    "myproject",
		DocRoots:       []string{"docs/", "README.md"},
		AnalyzeExclude: []string{"vendor/", "dist/"},
		HooksFile:      "hooks.yaml",
	}
	skill := hooks.GenerateSkill(cfg)

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
	if !strings.Contains(skill, cfg.HooksFile) {
		t.Errorf("expected skill to contain hooks file %q", cfg.HooksFile)
	}
}

func TestGenerateSkill_containsCoreSections(t *testing.T) {
	skill := hooks.GenerateSkill(hooks.GenerateConfig{
		TargetDir:   ".",
		ProjectName: "testproject",
		DocRoots:    []string{"docs/"},
		HooksFile:   "hooks.yaml",
	})

	requiredSections := []string{
		"## Selection Principle",
		"## Project Configuration",
		"## Process",
		"### Step 1",
		"### Step 2",
		"### Step 3",
		"### Step 4",
		"### Step 5",
		"### Step 6",
		"### Step 7",
		"## hooks.yaml Reference",
		"inject_inline",
		"PostToolUse",
		"PreToolUse",
	}
	for _, section := range requiredSections {
		if !strings.Contains(skill, section) {
			t.Errorf("expected skill to contain section %q", section)
		}
	}
}

func TestGenerateSkill_noDocRoots_formatsAsNone(t *testing.T) {
	skill := hooks.GenerateSkill(hooks.GenerateConfig{
		TargetDir:   ".",
		ProjectName: "testproject",
	})

	if !strings.Contains(skill, "**Doc corpus:** none") {
		t.Errorf("expected empty doc roots to format as 'none'")
	}
}

func TestGenerateSkill_noHooksFile_omitsField(t *testing.T) {
	skill := hooks.GenerateSkill(hooks.GenerateConfig{
		TargetDir:   ".",
		ProjectName: "testproject",
		DocRoots:    []string{"docs/"},
	})

	if strings.Contains(skill, "Existing hooks file") {
		t.Errorf("expected no hooks file reference when HooksFile is empty")
	}
}
