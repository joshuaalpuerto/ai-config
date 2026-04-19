package analyzer

import "time"

// AnalysisResult is the top-level output of the analyzer.
type AnalysisResult struct {
	Root              string              `json:"root"`
	AnalyzedAt        time.Time           `json:"analyzedAt"`
	GitChurnAvailable bool                `json:"gitChurnAvailable"`
	TechStack         TechStack           `json:"techStack"`
	TopLevelDirs      []string            `json:"topLevelDirs"`
	SourceFiles       []string            `json:"sourceFiles,omitempty"` // source file paths used for graph analysis, repo-relative
	AllFiles          []string            `json:"allFiles,omitempty"`    // all file paths in the repo, repo-relative
	Hubs              []Hub               `json:"hubs"`
	Hotspots          []Hotspot           `json:"hotspots"`
	Clusters          []Cluster           `json:"clusters"`
	Files             map[string]FileNode `json:"files,omitempty"`
}

// TechStack describes the detected languages and frameworks.
type TechStack struct {
	Languages  []string `json:"languages"`
	Frameworks []string `json:"frameworks"`
}

// FileNode holds per-file graph metrics.
type FileNode struct {
	Imports      []string `json:"imports"`
	ImportedBy   []string `json:"importedBy"`
	Lines        int      `json:"lines"`
	ExportCount  int      `json:"exportCount"`
	ExportNames  []string `json:"exportNames,omitempty"`
	Churn        int      `json:"churn"`
	FanIn        int      `json:"fanIn"`
	FanOut       int      `json:"fanOut"`
	IsEntryPoint bool     `json:"isEntryPoint"`
	Priority     float64  `json:"priority"`
	FolderDepth  int      `json:"folderDepth"`
}

// Hub is a high-priority file in the import graph.
type Hub struct {
	Path        string   `json:"path"`
	FanIn       int      `json:"fanIn"`
	FanOut      int      `json:"fanOut"`
	Priority    float64  `json:"priority"`
	ExportNames []string `json:"exportNames,omitempty"`
}

// Hotspot is a file with high git churn relative to its size.
type Hotspot struct {
	Path  string `json:"path"`
	Churn int    `json:"churn"`
	Lines int    `json:"lines"`
	Score int    `json:"score"`
}

// Cluster is a connected component in the import graph.
type Cluster struct {
	Label     string   `json:"label"`
	Size      int      `json:"size"`
	Singleton bool     `json:"singleton,omitempty"`
	DependsOn []string `json:"dependsOn,omitempty"` // labels of clusters this cluster imports from
	Files     []string `json:"files"`
}
