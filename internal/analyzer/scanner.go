package analyzer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// skipDirs are directory names that should never be walked.
var skipDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true,
	"dist": true, "build": true, "bin": true,
	"__pycache__": true, ".next": true, "coverage": true,
}

// srcExtensions are the file extensions treated as source files.
var srcExtensions = map[string]bool{
	".go": true, ".ts": true, ".tsx": true,
	".js": true, ".jsx": true, ".py": true,
}

// scanResult holds the output of the filesystem scan phase.
type scanResult struct {
	TechStack   TechStack
	SourceFiles []string          // absolute paths
	Domains     []string
	TSAliases   map[string]string // alias prefix → repo-relative dir
	ModulePath  string            // Go module path, if any
	SourceRoot  string            // absolute path to source root
}

// scan walks root and collects tech stack info, source files, and domain labels.
func scan(root string) (scanResult, error) {
	res := scanResult{
		TSAliases: make(map[string]string),
	}

	langSet := make(map[string]bool)
	fwSet := make(map[string]bool)

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil // skip unreadable entries
		}

		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		name := d.Name()

		// Marker file detection.
		switch name {
		case "go.mod":
			langSet["go"] = true
			if mod, err := parseGoMod(path); err == nil {
				res.ModulePath = mod
			}
		case "package.json":
			langSet["js/ts"] = true
			if fws, err := parsePackageJSON(path); err == nil {
				for _, fw := range fws {
					fwSet[fw] = true
				}
			}
		case "requirements.txt", "pyproject.toml":
			langSet["python"] = true
			if fws, err := parsePythonDeps(path); err == nil {
				for _, fw := range fws {
					fwSet[fw] = true
				}
			}
		case "tsconfig.json":
			if aliases, err := parseTSAliases(path, root); err == nil {
				for k, v := range aliases {
					res.TSAliases[k] = v
				}
			}
		}

		// Collect source files.
		ext := strings.ToLower(filepath.Ext(name))
		if srcExtensions[ext] {
			res.SourceFiles = append(res.SourceFiles, path)
		}
		return nil
	})
	if err != nil {
		return res, err
	}

	for lang := range langSet {
		res.TechStack.Languages = append(res.TechStack.Languages, lang)
	}
	for fw := range fwSet {
		res.TechStack.Frameworks = append(res.TechStack.Frameworks, fw)
	}

	res.SourceRoot = detectSourceRoot(root)
	res.Domains = discoverDomains(res.SourceRoot, root)

	return res, nil
}

// detectSourceRoot returns the first of src/, app/, lib/, server/ found at top level,
// or root itself as fallback.
func detectSourceRoot(root string) string {
	for _, name := range []string{"src", "app", "lib", "server"} {
		candidate := filepath.Join(root, name)
		if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
			return candidate
		}
	}
	return root
}

// discoverDomains returns the names of top-level directories inside sourceRoot
// that contain at least one source file (not in skipDirs).
func discoverDomains(sourceRoot, repoRoot string) []string {
	entries, err := os.ReadDir(sourceRoot)
	if err != nil {
		return nil
	}

	domainSet := make(map[string]bool)
	for _, e := range entries {
		if !e.IsDir() || skipDirs[e.Name()] {
			continue
		}
		dirPath := filepath.Join(sourceRoot, e.Name())
		if dirHasSourceFile(dirPath) {
			rel, err := filepath.Rel(repoRoot, dirPath)
			if err != nil {
				rel = e.Name()
			}
			domainSet[filepath.ToSlash(rel)] = true
		}
	}

	domains := make([]string, 0, len(domainSet))
	for d := range domainSet {
		domains = append(domains, d)
	}
	return domains
}

// dirHasSourceFile returns true if dir or any subdirectory contains a source file.
func dirHasSourceFile(dir string) bool {
	found := false
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if d.IsDir() && skipDirs[d.Name()] {
			return filepath.SkipDir
		}
		if !d.IsDir() {
			ext := strings.ToLower(filepath.Ext(d.Name()))
			if srcExtensions[ext] {
				found = true
			}
		}
		return nil
	})
	return found
}

// parseGoMod extracts the module path from a go.mod file.
func parseGoMod(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	return "", nil
}

// packageJSON is a minimal representation for framework detection.
type packageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

var jsFrameworks = map[string]string{
	"next":    "nextjs",
	"express": "express",
	"react":   "react",
	"vue":     "vue",
	"vite":    "vite",
}

// parsePackageJSON reads package.json and returns detected framework names.
func parsePackageJSON(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}

	fwSet := make(map[string]bool)
	for dep := range pkg.Dependencies {
		if fw, ok := jsFrameworks[dep]; ok {
			fwSet[fw] = true
		}
	}
	for dep := range pkg.DevDependencies {
		if fw, ok := jsFrameworks[dep]; ok {
			fwSet[fw] = true
		}
	}

	fws := make([]string, 0, len(fwSet))
	for fw := range fwSet {
		fws = append(fws, fw)
	}
	return fws, nil
}

var pythonFrameworks = []string{"fastapi", "django", "flask"}

// parsePythonDeps reads requirements.txt or pyproject.toml and returns detected frameworks.
func parsePythonDeps(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lower := strings.ToLower(string(data))

	var found []string
	for _, fw := range pythonFrameworks {
		if strings.Contains(lower, fw) {
			found = append(found, fw)
		}
	}
	return found, nil
}

// tsconfigPaths is a minimal representation of tsconfig.json for alias extraction.
type tsconfigPaths struct {
	CompilerOptions struct {
		Paths     map[string][]string `json:"paths"`
		BaseURL   string              `json:"baseUrl"`
	} `json:"compilerOptions"`
}

// parseTSAliases reads tsconfig.json and returns alias prefix → directory mappings.
func parseTSAliases(path, repoRoot string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg tsconfigPaths
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	result := make(map[string]string)
	tsconfigDir := filepath.Dir(path)
	for alias, targets := range cfg.CompilerOptions.Paths {
		if len(targets) == 0 {
			continue
		}
		// Strip trailing /* from alias key.
		prefix := strings.TrimSuffix(alias, "/*")
		// Resolve the first target directory relative to tsconfig location, then to repo root.
		target := strings.TrimSuffix(targets[0], "/*")
		abs := filepath.Join(tsconfigDir, target)
		rel, err := filepath.Rel(repoRoot, abs)
		if err != nil {
			continue
		}
		result[prefix] = filepath.ToSlash(rel)
	}
	return result, nil
}
