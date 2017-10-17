package main

/**
 * Load recent tweets from an account.
 */

import (
	"fmt"
	"log"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	ui "github.com/gizak/termui"
	"golang.org/x/oauth2"
)

////////////////////////////////////////////
// Util: Twitter
////////////////////////////////////////////

var twitterConfig = &oauth2.Config{}
var twitterToken = &oauth2.Token{AccessToken: GetTwitterAccessToken()}
var twitterHttpClient = twitterConfig.Client(oauth2.NoContext, twitterToken)
var twitterClient = twitter.NewClient(twitterHttpClient)

func newBool(myBool bool) *bool {
	b := myBool
	return &b
}

func GetLatestTweet(account string) string {
	log.Printf("Getting tweet for %v", account)
	tweets, _, err := twitterClient.Timelines.UserTimeline(&twitter.UserTimelineParams{
		ScreenName:      account,
		Count:           10,
		TrimUser:        newBool(true),
		ExcludeReplies:  newBool(true),
		IncludeRetweets: newBool(false),
	})

	if err != nil {
		log.Printf("Error loading tweets for '%v': %v", account, err)
	} else if len(tweets) < 1 {
		log.Printf("Failed to load any tweets for '%v'.", account)
	} else {
		t := tweets[0].FullText
		log.Printf("%v --> %v", account, t)
		return t
	}

	return "(no data)"
}

////////////////////////////////////////////
// Widget: Twitter
////////////////////////////////////////////

const TwitterWidgetUpdateInterval = 10 * time.Minute

type TwitterWidget struct {
	account     string
	widget      *ui.Par
	lastUpdated *time.Time
}

func NewTwitterWidget(account string) *TwitterWidget {
	// Create base element
	e := ui.NewPar("")
	e.Border = true
	e.BorderLabel = fmt.Sprintf("@%s", account)

	// Create widget
	w := &TwitterWidget{
		account: account,
		widget:  e,
	}

	w.update()
	w.resize()

	return w
}

func (w *TwitterWidget) getGridWidget() ui.GridBufferer {
	return w.widget
}

func (w *TwitterWidget) update() {
	if shouldUpdate(w) {
		// Get latest tweet
		w.widget.Text = GetLatestTweet(w.account)
	}
}

func (w *TwitterWidget) resize() {
	// Do nothing
}

func (w *TwitterWidget) getUpdateInterval() time.Duration {
	return TwitterWidgetUpdateInterval
}

func (w *TwitterWidget) getLastUpdated() *time.Time {
	return w.lastUpdated
}

func (w *TwitterWidget) setLastUpdated(t time.Time) {
	w.lastUpdated = &t
}
