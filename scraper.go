package scraper

import (
	"log"
	"strings"

	"github.com/pkg/errors"
)

// Scraper represents the scraper object
type Scraper struct {
	query     string
	total     uint32
	session   string
	fixed     string
	lastTweet *Tweet
}

func (s *Scraper) calculateMaxPosition(last string) *maxPosition {
	return &maxPosition{
		last:    last,
		session: s.session,
		fixed:   s.fixed,
	}
}
func (s *Scraper) loop(channel chan<- Tweet, last string) {
	// log.Printf("Loop for %s", last)

	mp := s.calculateMaxPosition(last)
	pageURL := twitterURL(s.query, mp)

	doc, newLast, err := getJSONPage(pageURL)
	if err != nil {
		myError := errors.Wrap(err, "Cannot get page for loop")
		log.Printf("Loop error %s", myError)
	}

	tweets := extractTweets(doc)
	// log.Printf("Got %d tweets", len(tweets))
	for _, tweet := range tweets {
		channel <- tweet
	}

	// Exit strategy
	if strings.Compare(newLast, last) == 0 {
		// log.Printf("No more data, bye")
		close(channel)
	} else {
		go s.loop(channel, newLast)
	}
}
func (s *Scraper) fetchSession() (*maxPosition, error) {
	// log.Printf("Getting session data")

	pageURL := twitterURL(s.query, nil)
	// log.Printf("Page url: %s", pageURL)

	doc, err := getHTMLPage(pageURL)
	if err != nil {
		return nil, errors.Wrap(err, "Cannot get page for session")
	}

	mpStr, ok := doc.Find(sessionSelector).Attr("data-max-position")
	if !ok {
		return nil, errors.Wrap(err, "Cannot get data-max-position")
	}
	mp := parseMaxPosition(mpStr)
	// log.Printf("Got max position: %s", mp)

	return mp, nil
}

// Start starts the scraping
func (s *Scraper) Start() (<-chan Tweet, error) {
	channel := make(chan Tweet, 0)
	// log.Printf("Starting scraper for: '%s'", s.query)

	mp, err := s.fetchSession()
	if err != nil {
		return nil, errors.Wrap(err, "Cannot get session data")
	}

	s.session = mp.session
	s.fixed = mp.fixed

	go s.loop(channel, mp.last)
	return channel, nil
}

// New creates a new scraper based on the passed query
func New(query string) (*Scraper, error) {
	query = strings.TrimSpace(query)
	if len(query) == 0 {
		return nil, errors.New("Query cannot be empty")
	}

	scraper := &Scraper{
		query:     query,
		total:     0,
		session:   "",
		fixed:     "",
		lastTweet: nil,
	}

	return scraper, nil
}
