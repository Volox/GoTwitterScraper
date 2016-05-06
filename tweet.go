package scraper

import (
	"strconv"

	"github.com/PuerkitoBio/goquery"
)

// Tweet represents a single tweet
type Tweet struct {
	ID        string
	Text      string
	Timestamp uint64
}

func extractTweets(doc *goquery.Document) []Tweet {
	var tweets []Tweet

	doc.Find(tweetsSelector).Each(func(i int, div *goquery.Selection) {
		id := div.AttrOr("data-item-id", "")
		text := div.Find(".tweet-text").Text()
		timestampData := doc.Find("._timestamp").AttrOr("data-time", "0")
		timestamp, err := strconv.ParseUint(timestampData, 10, 64)

		if len(id) != 0 || err == nil {
			// log.Printf("Adding tweet: %s", id)
			tweets = append(tweets, Tweet{
				ID:        id,
				Text:      text,
				Timestamp: timestamp,
			})
		}
	})

	return tweets
}
