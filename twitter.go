package main

/**
 * Load recent tweets from an account.
 */

import (
	"fmt"
	"log"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	ui "github.com/gizak/termui"
)

////////////////////////////////////////////
// Util: Twitter
////////////////////////////////////////////

var twitterConfig = oauth1.NewConfig(GetTwitterConsumerKey(), GetTwitterConsumerSecret())
var twitterToken = oauth1.NewToken(GetTwitterAccessToken(), GetTwitterAccessTokenSecret())
var twitterHttpClient = twitterConfig.Client(oauth1.NoContext, twitterToken)
var twitterClient = twitter.NewClient(twitterHttpClient)

func newBool(myBool bool) *bool {
	b := myBool
	return &b
}

func GetLatestTweet(account string) string {
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
		t := tweets[0].Text
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
	color       ui.Attribute
	widget      *ui.Par
	lastUpdated *time.Time
}

func NewTwitterWidget(account string, color ui.Attribute) *TwitterWidget {
	// Create base element
	e := ui.NewPar("")
	e.Border = true
	e.BorderLabel = fmt.Sprintf("@%s", account)
	e.BorderLabelFg = ui.ColorGreen
	e.TextFgColor = color

	// Create widget
	w := &TwitterWidget{
		account: account,
		color:   color,
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

	w.resize()
}

func (w *TwitterWidget) resize() {
	borderCount := 0
	if w.widget.Border {
		borderCount = 2
	}

	// Make line wrapping better
	wrap := w.widget.Width - borderCount
	if wrap <= 0 {
		wrap = 30
	}
	w.widget.WrapLength = wrap

	// Guess at line count
	height := borderCount + 1 + len(w.widget.Text)/wrap
	if height < 7 {
		height = 7
	}
	w.widget.Height = height
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
