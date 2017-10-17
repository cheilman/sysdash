package main

/**
 * Load configuration.  Right now that's all from the environment variables.  Maybe someday do something better?
 */

import (
	"log"
	"os"
	"strconv"
	"strings"
)

// TODO: Get these dynamically
var defaultGitRepoSearch = map[string]int{
	os.ExpandEnv("$HOME"): 3,
}

func GetGitRepoSearchPaths() map[string]int {
	repos := os.ExpandEnv("$SYSDASH_REPO_SEARCH_PATHS")

	if len(repos) <= 0 {
		return defaultGitRepoSearch
	} else {
		retval := make(map[string]int, 0)

		// Parse it out.  Current format is path:depth,path:depth,path:depth...
		pathDepths := strings.Split(repos, ",")

		for _, pathDepth := range pathDepths {
			parts := strings.Split(pathDepth, ":")

			if len(parts) != 2 {
				log.Printf("Error parsing pathDepth '%v'.  Part length: %d", pathDepth, len(parts))
			} else {
				path := parts[0]
				depth, depthErr := strconv.Atoi(parts[1])

				if depthErr != nil {
					log.Printf("Error converting depth part '%v': %v", parts[1], depthErr)
				} else {
					path = normalizePath(path)
					retval[path] = depth
				}
			}
		}

		if len(retval) <= 0 {
			log.Printf("Got no entries when parsing repos environment var: '%v'.  Using defaults.", repos)
			return defaultGitRepoSearch
		} else {
			return retval
		}
	}
}
