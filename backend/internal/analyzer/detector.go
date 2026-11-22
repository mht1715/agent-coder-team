package analyzer

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"backend/internal/domain"
)

var ignoreDirs = map[string]struct{}{
	".git": {}, ".idea": {}, ".vscode": {}, ".next": {}, ".nuxt": {},
	"node_modules": {}, "vendor": {}, "dist": {}, "build": {}, "target": {},
	"bin": {}, "obj": {}, "coverage": {}, "tmp": {}, ".venv": {}, "venv": {},
	"__pycache__": {}, ".pytest_cache": {}, ".mypy_cache": {},
}

var languageByExt = map[string]string{
	".go": "Go", ".js": "JavaScript", ".jsx": "JavaScript",
	".ts": "TypeScript", ".tsx": "TypeScript", ".vue": "Vue",
	".py": "Python", ".java": "Java", ".kt": "Kotlin",
	".cs": "C#", ".rs": "Rust", ".rb": "Ruby", ".php": "PHP",
}

type packageJSON struct {
	Scripts         map[string]string `json:"scripts"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

type Detector struct {
	maxFiles int
}

func NewDetector(maxFiles int) *Detector {
	if maxFiles <= 0 {
		maxFiles = 20000
	}
	return &Detector{maxFiles: maxFiles}
}

func (d *Detector) Detect(root string) (*domain.ProjectProfile, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("stat project directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("project directory is not a directory")
	}

	now := time.Now().UTC()
	profile := &domain.ProjectProfile{
		Languages:  []string{},
		Frameworks: []string{},
		TopLevel:   []string{},
		DetectedAt: now,
		UpdatedAt:  now,
	}

	languageCounts := map[string]int{}
	topLevel := map[string]struct{}{}
	fileCount := 0
	truncated := false

	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if path == root {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		depth := len(strings.Split(rel, string(filepath.Separator)))

		if entry.IsDir() {
			if depth == 1 && !strings.HasPrefix(entry.Name(), ".") {
				topLevel[entry.Name()] = struct{}{}
			}
			if _, ignored := ignoreDirs[entry.Name()]; ignored {
				return fs.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}

		fileCount++
		if fileCount > d.maxFiles {
			truncated = true
			return fs.SkipAll
		}
		if language, ok := languageByExt[strings.ToLower(filepath.Ext(entry.Name()))]; ok {
			languageCounts[language]++
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan project directory: %w", err)
	}

	profile.FileCount = min(fileCount, d.maxFiles)
	profile.Truncated = truncated
	for name := range topLevel {
		profile.TopLevel = append(profile.TopLevel, name)
	}
	sort.Strings(profile.TopLevel)

	profile.Languages = rankedLanguages(languageCounts)
	if len(profile.Languages) > 0 {
		profile.PrimaryLanguage = profile.Languages[0]
	}

	d.detectGo(root, profile)
	d.detectNode(root, profile, languageCounts)
	d.detectPython(root, profile)
	d.detectJava(root, profile)
	d.detectDotNet(root, profile)
	d.detectRust(root, profile)
	d.detectUniApp(root, profile)

	profile.Frameworks = uniqueStrings(profile.Frameworks)
	profile.Languages = rankedLanguages(languageCounts)
	if profile.PrimaryLanguage == "" && len(profile.Languages) > 0 {
		profile.PrimaryLanguage = profile.Languages[0]
	}
	if profile.PrimaryLanguage == "" {
		profile.PrimaryLanguage = "Unknown"
	}
	return profile, nil
}

func (d *Detector) detectGo(root string, profile *domain.ProjectProfile) {
	content, ok := readSmallFile(filepath.Join(root, "go.mod"))
	if !ok {
		return
	}
	profile.Languages = append(profile.Languages, "Go")
	profile.PackageManager = "go modules"
	profile.BuildCommand = "go build ./..."
	profile.TestCommand = "go test ./..."
	profile.FormatCommand = "gofmt -w ."
	if strings.Contains(content, "gin-gonic/gin") {
		profile.Frameworks = append(profile.Frameworks, "Gin")
	}
	if strings.Contains(content, "gorm.io/gorm") {
		profile.Frameworks = append(profile.Frameworks, "GORM")
	}
	if strings.Contains(content, "spf13/viper") {
		profile.Frameworks = append(profile.Frameworks, "Viper")
	}
}

func (d *Detector) detectNode(root string, profile *domain.ProjectProfile, languageCounts map[string]int) {
	content, ok := readSmallFile(filepath.Join(root, "package.json"))
	if !ok {
		return
	}
	languageCounts["JavaScript"]++

	var pkg packageJSON
	if err := json.Unmarshal([]byte(content), &pkg); err != nil {
		return
	}
	pm := "npm"
	switch {
	case fileExists(filepath.Join(root, "pnpm-lock.yaml")):
		pm = "pnpm"
	case fileExists(filepath.Join(root, "yarn.lock")):
		pm = "yarn"
	case fileExists(filepath.Join(root, "bun.lockb")):
		pm = "bun"
	}
	profile.PackageManager = pm
	if _, ok := pkg.Scripts["build"]; ok {
		profile.BuildCommand = pm + " run build"
	}
	if _, ok := pkg.Scripts["test"]; ok {
		profile.TestCommand = pm + " run test"
	}
	if _, ok := pkg.Scripts["lint"]; ok {
		profile.FormatCommand = pm + " run lint"
	}

	deps := map[string]string{}
	for k, v := range pkg.Dependencies {
		deps[k] = v
	}
	for k, v := range pkg.DevDependencies {
		deps[k] = v
	}
	addFramework := func(key, name string) {
		if _, ok := deps[key]; ok {
			profile.Frameworks = append(profile.Frameworks, name)
		}
	}
	addFramework("vue", "Vue")
	addFramework("react", "React")
	addFramework("next", "Next.js")
	addFramework("vite", "Vite")
	addFramework("nuxt", "Nuxt")
	addFramework("@nestjs/core", "NestJS")
	addFramework("express", "Express")
	addFramework("@dcloudio/uni-app", "uni-app")
}

func (d *Detector) detectPython(root string, profile *domain.ProjectProfile) {
	content := ""
	if value, ok := readSmallFile(filepath.Join(root, "pyproject.toml")); ok {
		content += value
	}
	if value, ok := readSmallFile(filepath.Join(root, "requirements.txt")); ok {
		content += value
	}
	if content == "" {
		return
	}
	profile.Languages = append(profile.Languages, "Python")
	profile.PackageManager = "pip"
	if fileExists(filepath.Join(root, "uv.lock")) {
		profile.PackageManager = "uv"
	} else if fileExists(filepath.Join(root, "poetry.lock")) {
		profile.PackageManager = "poetry"
	}
	profile.TestCommand = "python -m pytest"
	profile.FormatCommand = "ruff format ."
	lower := strings.ToLower(content)
	if strings.Contains(lower, "fastapi") {
		profile.Frameworks = append(profile.Frameworks, "FastAPI")
	}
	if strings.Contains(lower, "django") {
		profile.Frameworks = append(profile.Frameworks, "Django")
	}
	if strings.Contains(lower, "flask") {
		profile.Frameworks = append(profile.Frameworks, "Flask")
	}
}

func (d *Detector) detectJava(root string, profile *domain.ProjectProfile) {
	if fileExists(filepath.Join(root, "pom.xml")) {
		profile.Languages = append(profile.Languages, "Java")
		profile.PackageManager = "maven"
		profile.BuildCommand = "mvn -q -DskipTests package"
		profile.TestCommand = "mvn test"
		if content, ok := readSmallFile(filepath.Join(root, "pom.xml")); ok && strings.Contains(content, "spring-boot") {
			profile.Frameworks = append(profile.Frameworks, "Spring Boot")
		}
		return
	}
	if fileExists(filepath.Join(root, "build.gradle")) || fileExists(filepath.Join(root, "build.gradle.kts")) {
		profile.Languages = append(profile.Languages, "Java")
		profile.PackageManager = "gradle"
		if fileExists(filepath.Join(root, "gradlew")) {
			profile.BuildCommand = "./gradlew build"
			profile.TestCommand = "./gradlew test"
		} else {
			profile.BuildCommand = "gradle build"
			profile.TestCommand = "gradle test"
		}
	}
}

func (d *Detector) detectDotNet(root string, profile *domain.ProjectProfile) {
	if hasFileWithExt(root, ".csproj") || hasFileWithExt(root, ".sln") {
		profile.Languages = append(profile.Languages, "C#")
		profile.PackageManager = "nuget"
		profile.BuildCommand = "dotnet build"
		profile.TestCommand = "dotnet test"
		profile.FormatCommand = "dotnet format"
		profile.Frameworks = append(profile.Frameworks, ".NET")
	}
}

func (d *Detector) detectRust(root string, profile *domain.ProjectProfile) {
	if fileExists(filepath.Join(root, "Cargo.toml")) {
		profile.Languages = append(profile.Languages, "Rust")
		profile.PackageManager = "cargo"
		profile.BuildCommand = "cargo build"
		profile.TestCommand = "cargo test"
		profile.FormatCommand = "cargo fmt"
	}
}

func (d *Detector) detectUniApp(root string, profile *domain.ProjectProfile) {
	if fileExists(filepath.Join(root, "manifest.json")) && fileExists(filepath.Join(root, "pages.json")) {
		profile.Frameworks = append(profile.Frameworks, "uni-app")
		if profile.BuildCommand == "" {
			profile.BuildCommand = "pnpm build:h5"
		}
		if profile.TestCommand == "" {
			profile.TestCommand = "pnpm test"
		}
	}
}

func readSmallFile(path string) (string, bool) {
	info, err := os.Stat(path)
	if err != nil || info.Size() > 1<<20 {
		return "", false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return string(data), true
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func hasFileWithExt(root, ext string) bool {
	found := false
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if entry.IsDir() {
			if _, ignored := ignoreDirs[entry.Name()]; ignored {
				return fs.SkipDir
			}
			return nil
		}
		if strings.EqualFold(filepath.Ext(entry.Name()), ext) {
			found = true
			return fs.SkipAll
		}
		return nil
	})
	return found
}

func rankedLanguages(counts map[string]int) []string {
	type languageCount struct {
		name  string
		count int
	}
	items := make([]languageCount, 0, len(counts))
	for name, count := range counts {
		if count > 0 {
			items = append(items, languageCount{name: name, count: count})
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].count == items[j].count {
			return items[i].name < items[j].name
		}
		return items[i].count > items[j].count
	})
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.name)
	}
	return result
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
