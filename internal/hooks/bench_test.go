package hooks

import (
	"testing"
)

// Benchmarks comparing old extension/directory matching vs new doublestar-based paths matching.

// --- Old approach (inlined for comparison) ---

func oldExtensionMatches(filePath string, extensions []string) bool {
	ext := ""
	for i := len(filePath) - 1; i >= 0; i-- {
		if filePath[i] == '.' {
			ext = filePath[i:]
			break
		}
	}
	for _, e := range extensions {
		if e == ext {
			return true
		}
	}
	return false
}

func oldDirectoryPrefix(filePath string, dirs []string) bool {
	for _, d := range dirs {
		if len(filePath) > len(d) && filePath[:len(d)] == d && filePath[len(d)] == '/' {
			return true
		}
	}
	return false
}

// --- Benchmarks ---

var benchPaths = []string{
	"/home/user/project/src/api/routes.py",
	"/home/user/project/src/api/deep/nested/handler.py",
	"/home/user/project/tests/test_routes.ts",
	"/home/user/project/README.md",
}

func BenchmarkOld_ExtensionMatch(b *testing.B) {
	exts := []string{".py", ".pyi"}
	for i := 0; i < b.N; i++ {
		for _, p := range benchPaths {
			oldExtensionMatches(p, exts)
		}
	}
}

func BenchmarkNew_PathsMatch_ExtensionGlob(b *testing.B) {
	patterns := []string{"*.py", "*.pyi"}
	for i := 0; i < b.N; i++ {
		for _, p := range benchPaths {
			pathsMatch(p, patterns)
		}
	}
}

func BenchmarkOld_DirectoryPrefix(b *testing.B) {
	dirs := []string{"src/api"}
	for i := 0; i < b.N; i++ {
		for _, p := range benchPaths {
			oldDirectoryPrefix(p, dirs)
		}
	}
}

func BenchmarkNew_PathsMatch_DirectorySlash(b *testing.B) {
	patterns := []string{"src/api/"}
	for i := 0; i < b.N; i++ {
		for _, p := range benchPaths {
			pathsMatch(p, patterns)
		}
	}
}

func BenchmarkNew_PathsMatch_DoubleStarExt(b *testing.B) {
	patterns := []string{"src/api/**/*.py"}
	for i := 0; i < b.N; i++ {
		for _, p := range benchPaths {
			pathsMatch(p, patterns)
		}
	}
}

func BenchmarkNew_PathsMatch_MultiplePatterns(b *testing.B) {
	patterns := []string{"*.py", "*.pyi", "src/api/**", "routes/**"}
	for i := 0; i < b.N; i++ {
		for _, p := range benchPaths {
			pathsMatch(p, patterns)
		}
	}
}

func BenchmarkFullEvaluate_SingleRule_3Patterns(b *testing.B) {
	cfg := HooksConfig{
		PreToolUse: []Rule{
			{
				Match:  Matchers{Tools: []string{"Write"}, Paths: []string{"*.py", "*.pyi"}},
				Action: Actions{InjectInline: "follow PEP8"},
			},
		},
	}
	event := fileEvent("Write", "/home/user/project/src/api/routes.py")
	for i := 0; i < b.N; i++ {
		Evaluate(event, cfg, nil)
	}
}

func BenchmarkFullEvaluate_5Rules_MixedPatterns(b *testing.B) {
	cfg := HooksConfig{
		PreToolUse: []Rule{
			{
				Match:  Matchers{Tools: []string{"Write"}, Paths: []string{"*.py", "*.pyi"}},
				Action: Actions{InjectInline: "python standards"},
			},
			{
				Match:  Matchers{Tools: []string{"Write"}, Paths: []string{"src/api/**"}},
				Action: Actions{InjectInline: "api guidelines"},
			},
			{
				Match:  Matchers{Tools: []string{"Write"}, Paths: []string{".env"}},
				Action: Actions{Block: boolPtr(true)},
			},
			{
				Match:  Matchers{Tools: []string{"Write"}, Paths: []string{"*.ts", "*.tsx"}},
				Action: Actions{InjectInline: "ts standards"},
			},
			{
				Match:  Matchers{Tools: []string{"Write"}, Paths: []string{"src/**/nested/"}},
				Action: Actions{InjectInline: "nested dir warning"},
			},
		},
	}
	event := fileEvent("Write", "/home/user/project/src/api/routes.py")
	for i := 0; i < b.N; i++ {
		Evaluate(event, cfg, nil)
	}
}
