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

////////////////////////////////////////////
// Debugging
////////////////////////////////////////////

func LogToFile() bool {
	tofile := os.ExpandEnv("$SYSDASH_LOG_TO_FILE")

	if len(tofile) > 0 {
		dolog, err := strconv.ParseBool(tofile)

		if err != nil {
			log.Printf("Failed to parse '%v' from SYSDASH_LOG_TO_FILE: %v", tofile, err)
		} else {
			return dolog
		}
	}

	return false
}

////////////////////////////////////////////
// Git Repos
////////////////////////////////////////////

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

////////////////////////////////////////////
// Twitter Keys
////////////////////////////////////////////

const DefaultTwitter1 = "tinycarebot"
const DefaultTwitter2 = "selfcare_bot"
const DefaultTwitter3 = "str_voyage"

func GetTwitterAccount1() string {
	acct := os.ExpandEnv("$SYSDASH_TWITTER_ACCT_1")

	if len(acct) <= 0 {
		return DefaultTwitter1
	} else {
		return acct
	}
}

func GetTwitterAccount2() string {
	acct := os.ExpandEnv("$SYSDASH_TWITTER_ACCT_2")

	if len(acct) <= 0 {
		return DefaultTwitter2
	} else {
		return acct
	}
}

func GetTwitterAccount3() string {
	acct := os.ExpandEnv("$SYSDASH_TWITTER_ACCT_3")

	if len(acct) <= 0 {
		return DefaultTwitter3
	} else {
		return acct
	}
}

func GetTwitterConsumerKey() string {
	return os.ExpandEnv("$SYSDASH_TWITTER_CONSUMER_KEY")
}

func GetTwitterConsumerSecret() string {
	return os.ExpandEnv("$SYSDASH_TWITTER_CONSUMER_SECRET")
}

func GetTwitterAccessToken() string {
	return os.ExpandEnv("$SYSDASH_TWITTER_ACCESS_TOKEN")
}

func GetTwitterAccessTokenSecret() string {
	return os.ExpandEnv("$SYSDASH_TWITTER_ACCESS_TOKEN_SECRET")
}

////////////////////////////////////////////
// Weather
////////////////////////////////////////////

const DefaultWeatherLocation = "Pittsburgh,PA"

func GetWeatherLocation() string {
	loc := os.ExpandEnv("$SYSDASH_WEATHER_LOCATION")

	if len(loc) <= 0 {
		return DefaultWeatherLocation
	} else {
		return loc
	}
}
