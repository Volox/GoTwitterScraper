package scraper

import "fmt"
import "strings"

type maxPosition struct {
	session string
	fixed   string
	last    string
}

// String implements the stringer interfce
func (mp maxPosition) String() string {
	// TWEET-<numbers>-<numbers>-<sessionId>
	return fmt.Sprintf("TWEET-%s-%s-%s", mp.last, mp.fixed, mp.session)
}

func parseMaxPosition(str string) *maxPosition {
	parts := strings.Split(str, "-")
	return &maxPosition{
		session: parts[3],
		fixed:   parts[2],
		last:    parts[1],
	}
}
