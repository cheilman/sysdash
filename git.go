package main

/**
 * Git Repo Information
 */

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	ui "github.com/gizak/termui"
	walk "github.com/karrick/godirwalk"
)

////////////////////////////////////////////
// Utility: Git Repo Info
////////////////////////////////////////////

const GitRepoStatusUpdateInterval = 10 * time.Second

type RepoStatusField struct {
	OutputCharacter   rune
	OutputColorString string
}

// Key is the git status rune (what shows up in `git status -sb`)
var RepoStatusFieldDefinitionsOrderedKeys = []rune{'M', 'A', 'D', 'R', 'C', 'U', '?', '!'}
var RepoStatusFieldDefinitions = map[rune]RepoStatusField{
	// modified
	'M': RepoStatusField{OutputCharacter: 'M', OutputColorString: "fg-green"},
	// added
	'A': RepoStatusField{OutputCharacter: '+', OutputColorString: "fg-green,fg-bold"},
	// deleted
	'D': RepoStatusField{OutputCharacter: '-', OutputColorString: "fg-red,fg-bold"},
	// renamed
	'R': RepoStatusField{OutputCharacter: 'R', OutputColorString: "fg-yellow,fg-bold"},
	// copied
	'C': RepoStatusField{OutputCharacter: 'C', OutputColorString: "fg-blue,fg-bold"},
	// updated
	'U': RepoStatusField{OutputCharacter: 'U', OutputColorString: "fg-magenta,fg-bold"},
	// untracked
	'?': RepoStatusField{OutputCharacter: '?', OutputColorString: "fg-red"},
	// ignored
	'!': RepoStatusField{OutputCharacter: '!', OutputColorString: "fg-cyan"},
}

type RepoInfo struct {
	Name         string
	FullPath     string
	HomePath     string
	BranchStatus string
	Status       string
	lastUpdated  *time.Time
}

func NewRepoInfo(fullPath string) RepoInfo {
	if strings.HasSuffix(fullPath, ".git") || strings.HasSuffix(fullPath, ".git/") {
		// This is the path to the .git folder, so go up a level
		fullPath = normalizePath(filepath.Join(fullPath, ".."))
	}

	// Repo name
	name := filepath.Base(fullPath)

	// Normalize path with home directory (if possible)
	homePath := fullPath

	if strings.HasPrefix(fullPath, HOME) {
		relative, relErr := filepath.Rel(HOME, fullPath)

		if relErr == nil {
			homePath = filepath.Join("~", relative)
		} else {
			log.Printf("Error getting relative: %v", relErr)
		}
	} else if strings.HasPrefix(fullPath, CANONHOME) {
		relative, relErr := filepath.Rel(CANONHOME, fullPath)

		if relErr == nil {
			homePath = filepath.Join("~", relative)
		} else {
			log.Printf("Error getting relative: %v", relErr)
		}
	}

	// Load repo status
	branches := "my branches"
	status := "my status"

	// Build it
	r := RepoInfo{
		Name:         name,
		FullPath:     fullPath,
		HomePath:     homePath,
		BranchStatus: branches,
		Status:       status,
	}

	r.update()

	return r
}

func (w *RepoInfo) update() {
	if shouldUpdate(w) {
		// TODO: Make this not run a command to get this data
		// Go do a git status in that folder
		output, exitCode, err := execAndGetOutput("git", &w.FullPath, "-c", "color.status=never", "-c", "color.ui=never", "status", "-sb")

		if err != nil {
			log.Printf("Failed to get git output for repo %v (%v): %v", w.Name, w.FullPath, err)
		} else if exitCode != 0 {
			log.Printf("Bad exit code getting git output for repo %v (%v): %v", w.Name, w.FullPath, exitCode)
		} else {
			// Parse out the output
			lines := strings.Split(output, "\n")

			// Branch is first line
			branchLine := lines[0][3:]
			branchName := strings.Split(branchLine, " ")[0]
			if strings.Contains(branchName, "...") {
				branchName = strings.Split(branchName, "...")[0]
			}

			branchState := ""
			if strings.Contains(branchLine, "[") {
				branchState = "[" + strings.Split(branchLine, "[")[1]
			}

			nameColor := "fg-cyan"

			if branchName == "master" || branchName == "mainline" {
				nameColor = "fg-green"
			}

			w.BranchStatus = fmt.Sprintf("[%v](%s)", branchName, nameColor)

			if len(branchState) > 0 {
				w.BranchStatus += fmt.Sprintf(" [%v](fg-magenta)", branchState)
			}

			// Status for files follows, let's aggregate
			status := make(map[rune]int, len(RepoStatusFieldDefinitions))
			for field, _ := range RepoStatusFieldDefinitions {
				status[field] = 0
			}

			for _, l := range lines[1:] {
				l = strings.TrimSpace(l)

				if len(l) < 2 {
					continue
				}

				// Grab first two characters
				statchars := l[:2]

				for key := range status {
					if strings.ContainsRune(statchars, key) {
						status[key]++
					}
				}
			}

			w.Status = buildColoredStatusStringFromMap(status)
		}
	}
}

func (w *RepoInfo) getUpdateInterval() time.Duration {
	return GitRepoStatusUpdateInterval
}

func (w *RepoInfo) getLastUpdated() *time.Time {
	return w.lastUpdated
}

func (w *RepoInfo) setLastUpdated(t time.Time) {
	w.lastUpdated = &t
}

type BySortOrder []*ui.Gauge

func (a BySortOrder) Len() int           { return len(a) }
func (a BySortOrder) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a BySortOrder) Less(i, j int) bool { return a[i].BorderLabel < a[j].BorderLabel }

func buildColoredStatusStringFromMap(status map[rune]int) string {
	retval := ""

	for _, key := range RepoStatusFieldDefinitionsOrderedKeys {
		count := status[key]

		if count > 0 {
			if retval != "" {
				retval += " "
			}

			retval += fmt.Sprintf("[%c:%d](%s)", RepoStatusFieldDefinitions[key].OutputCharacter, count, RepoStatusFieldDefinitions[key].OutputColorString)
		}
	}

	return retval
}

////////////////////////////////////////////
// Utility: Git Repo List
////////////////////////////////////////////

const GitRepoListUpdateInterval = 30 * time.Second

var HOME = os.ExpandEnv("$HOME")
var CANONHOME = normalizePath(HOME)

type CachedGitRepoList struct {
	repoSearch  map[string]int
	Repos       []RepoInfo
	lastUpdated *time.Time
}

func (w *CachedGitRepoList) getUpdateInterval() time.Duration {
	return GitRepoListUpdateInterval
}

func (w *CachedGitRepoList) getLastUpdated() *time.Time {
	return w.lastUpdated
}

func (w *CachedGitRepoList) setLastUpdated(t time.Time) {
	w.lastUpdated = &t
}

func (w *CachedGitRepoList) update() {
	if shouldUpdate(w) {
		repoPaths := getGitRepositories(w.repoSearch)

		repos := make([]RepoInfo, 0)

		for _, repo := range repoPaths {
			repoInfo := NewRepoInfo(repo)

			repos = append(repos, repoInfo)
		}

		w.Repos = repos
	}

	// Update status for all the repos as well
	for _, r := range w.Repos {
		r.update()
	}
}

func NewCachedGitRepoList(search map[string]int) *CachedGitRepoList {
	// Build it
	w := &CachedGitRepoList{
		repoSearch: search,
		Repos:      make([]RepoInfo, 0),
	}

	w.update()

	return w
}

var cachedGitRepos = NewCachedGitRepoList(GetGitRepoSearchPaths())

// Walks the search directories to look for git folders
// search is a map of directory roots to depths
func getGitRepositories(search map[string]int) []string {
	var retval = make([]string, 0)

	for path, depth := range search {
		gitRepos := getGitRepositoriesForPath(path, depth)

		retval = append(retval, gitRepos...)
	}

	// Sort
	sort.Strings(retval)

	// Uniquify
	// w is where non-matching elements should be written
	// last is the last element we wrote
	// r is the current read pointer
	w := 1
	last := 0
	for r := 1; r < len(retval); r++ {
		// If they're the same, skip it
		if retval[r] == retval[last] {
			continue
		}

		// They're different, write it to the array
		retval[w] = retval[r]

		// Save last pointer
		last = w

		// Advance
		w++
	}

	retval = retval[:w] // slice it to just what we wrote

	return retval
}

func getGitRepositoriesForPath(root string, maxDepth int) []string {
	var retval = walkTreeLookingForGit(root, nil, 0, maxDepth)

	return retval
}

func walkTreeLookingForGit(path string, de *walk.Dirent, curDepth int, maxDepth int) []string {
	// Do we keep going?
	if curDepth <= maxDepth {
		// de is nil the first time through
		if de != nil {
			gitPath := checkAndResolveGitFolder(path, de)

			if gitPath != nil {
				// Got it!
				return []string{*gitPath}
			}
		}

		// Get children
		retval := make([]string, 0)

		kids, err := walk.ReadDirents(path, nil)

		if err != nil {
			log.Printf("Failed to traverse into children of '%v': %v", path, err)
		} else {
			for _, kidDE := range kids {
				if kidDE.IsDir() {
					results := walkTreeLookingForGit(filepath.Join(path, kidDE.Name()), kidDE, curDepth+1, maxDepth)

					retval = append(retval, results...)
				}
			}
		}

		return retval
	} else {
		return []string{}
	}
}

// Returns nil if not a git folder
// Returns a resolved pathname if is a git folder
func checkAndResolveGitFolder(osPathname string, de *walk.Dirent) *string {
	// check name
	if !de.IsDir() {
		return nil
	}

	if de.Name() != ".git" {
		return nil
	}

	path := normalizePath(osPathname)
	return &path
}

////////////////////////////////////////////
// Widget: Git Repos
////////////////////////////////////////////

const MinimumRepoNameWidth = 26
const MinimumRepoBranchesWidth = 37

type GitRepoWidget struct {
	widget      *ui.Table
	lastUpdated *time.Time
}

func NewGitRepoWidget() *GitRepoWidget {
	// Create base element
	e := ui.NewTable()
	e.Border = true
	e.BorderLabel = "Git Repos"
	e.Separator = false

	// Create widget
	w := &GitRepoWidget{
		widget: e,
	}

	w.update()
	w.resize()

	return w
}

func (w *GitRepoWidget) getGridWidget() ui.GridBufferer {
	return w.widget
}

func (w *GitRepoWidget) update() {
	rows := [][]string{}
	height := 2

	// Load repos
	cachedGitRepos.update()

	maxRepoWidth := 0

	for _, repo := range cachedGitRepos.Repos {
		// Figure out max length
		if len(repo.HomePath) > maxRepoWidth {
			maxRepoWidth = len(repo.HomePath)
		}
	}

	if maxRepoWidth < MinimumRepoNameWidth {
		maxRepoWidth = MinimumRepoNameWidth
	}

	for _, repo := range cachedGitRepos.Repos {
		// Make the name all fancy
		pathPad := maxRepoWidth - len(repo.Name)
		path := filepath.Dir(repo.HomePath)

		name := fmt.Sprintf("[%*v%c](fg-cyan)[%v](fg-cyan,fg-bold)", pathPad, path, os.PathSeparator, repo.Name)

		line := []string{name, repo.BranchStatus, repo.Status}

		rows = append(rows, line)
		height++
	}

	w.widget.Rows = rows
	w.widget.Height = height

}

func (w *GitRepoWidget) resize() {
	// Do nothing
}
