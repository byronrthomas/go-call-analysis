package analyzer

import (
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// GitRevisionCache provides thread-safe caching of git revision information for files
type GitRevisionCache struct {
	cache      map[string]string
	mutex      sync.RWMutex
	projectDir string
}

// NewGitRevisionCache creates a new git revision cache
func NewGitRevisionCache(projectDir string) *GitRevisionCache {
	return &GitRevisionCache{
		cache:      make(map[string]string),
		projectDir: projectDir,
	}
}

// GetFileRevision returns the latest git commit hash for a given file
func (g *GitRevisionCache) GetFileRevision(filename string) string {
	if filename == "" {
		return ""
	}

	// Check cache first
	g.mutex.RLock()
	if revision, exists := g.cache[filename]; exists {
		g.mutex.RUnlock()
		return revision
	}
	g.mutex.RUnlock()

	// Get revision from git
	revision := g.fetchGitRevision(filename)

	// Cache the result
	g.mutex.Lock()
	g.cache[filename] = revision
	g.mutex.Unlock()

	return revision
}

// fetchGitRevision executes git command to get the latest commit hash for a file
func (g *GitRevisionCache) fetchGitRevision(filename string) string {
	// Convert to relative path if it's absolute and within project directory
	relPath := filename
	if filepath.IsAbs(filename) {
		if rel, err := filepath.Rel(g.projectDir, filename); err == nil && !strings.HasPrefix(rel, "..") {
			relPath = rel
		}
	}

	// Execute git log command to get latest commit hash for the file
	cmd := exec.Command("git", "log", "-1", "--format=%H", "--", relPath)
	cmd.Dir = g.projectDir

	output, err := cmd.Output()
	if err != nil {
		log.Printf("WARN: Failed to get git revision for file %s: %v", filename, err)
		return ""
	}

	revision := strings.TrimSpace(string(output))
	return revision
}
