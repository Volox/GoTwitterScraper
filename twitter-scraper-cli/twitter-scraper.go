package main

import (
	"log"
	"net/url"
	"os"

	"fmt"

	"strings"

	"strconv"

	"time"

	"github.com/ChimeraCoder/anaconda"
	"github.com/Volox/GoTwitterScraper"
	"github.com/codegangsta/cli"
	"gopkg.in/mgo.v2"
)

type location struct {
	Type        string
	Coordinates []float64
}
type tweet struct {
	Source    string
	ID        string `bson:"id"`
	Text      string
	Date      time.Time
	Timestamp int64
	Location  *location
	Author    string
	AuthorID  string `bson:"authorId"`
	Tags      []string
	Lang      string
}

func convertRawTweet(raw anaconda.Tweet) *tweet {
	date, err := raw.CreatedAtTime()
	if err != nil {
		log.Printf("Cannot convert date")
		return nil
	}

	var tags []string
	for _, tag := range raw.Entities.Hashtags {
		tags = append(tags, tag.Text)
	}

	t := tweet{
		Source:    "twitter",
		ID:        raw.IdStr,
		Text:      raw.Text,
		Date:      date,
		Timestamp: date.Unix() * 1000,
		Author:    raw.User.ScreenName,
		AuthorID:  raw.User.IdStr,
		Tags:      tags,
		Lang:      raw.Lang,
	}

	if raw.HasCoordinates() {
		longitude, err := raw.Longitude()
		if err != nil {
			goto end
		}
		latitude, err := raw.Latitude()
		if err != nil {
			goto end
		}

		t.Location = &location{
			Type: "Point",
			Coordinates: []float64{
				longitude,
				latitude,
			},
		}
	}

end:
	return &t
}

func configureTwitter(ctx *cli.Context, in <-chan interface{}) (<-chan interface{}, error) {
	log.Printf("Twitter configured")
	out := make(chan interface{})

	key := ctx.String("key")
	secret := ctx.String("secret")
	token := ctx.String("token")
	tokenSecret := ctx.String("token-secret")
	anaconda.SetConsumerKey(key)
	anaconda.SetConsumerSecret(secret)
	api := anaconda.NewTwitterApi(token, tokenSecret)

	go func() {
		defer close(out)

		ids := make([]int64, 0, 100)
		for element := range in {
			origTweet := element.(scraper.Tweet)
			id, err := strconv.ParseInt(origTweet.ID, 10, 64)
			if err != nil {
				log.Printf("Cannot get id, skip")
				continue
			}
			ids = append(ids, id)

			if len(ids) == 100 {
				tweets, err := api.GetTweetsLookupByIds(ids, url.Values{})
				if err != nil {
					log.Printf("Cannot get tweets by IDS")
					continue
				}

				for _, rawTweet := range tweets {
					tweet := convertRawTweet(rawTweet)
					if tweet != nil {
						out <- tweet
					}
				}

				ids = ids[:0]
			}
		}
	}()

	return out, nil
}
func configureMongo(ctx *cli.Context, in <-chan interface{}) (<-chan interface{}, error) {
	log.Printf("Mongo configured")
	out := make(chan interface{})

	host := ctx.String("host")
	port := ctx.Int("port")
	database := ctx.String("database")
	collection := ctx.String("collection")

	mongoURL := url.URL{
		Scheme: "mongodb",
		Host:   fmt.Sprintf("%s:%d", host, port),
		Path:   database,
	}

	log.Printf("MongoURL: %s", mongoURL.String())
	log.Printf("Collection: %s", collection)

	session, err := mgo.Dial(mongoURL.String())
	if err != nil {
		return nil, err
	}
	c := session.DB(database).C(collection)

	go func() {
		defer close(out)

		for element := range in {
			err := c.Insert(element)
			if err != nil {
				log.Printf("Cannot insert tweet %v", err)
			}
		}

	}()

	return out, nil
}
func generalize(in <-chan scraper.Tweet) <-chan interface{} {
	out := make(chan interface{})

	go func() {
		defer close(out)

		for tweet := range in {
			// log.Printf("Converting: %s", tweet.ID)
			out <- tweet
		}
	}()

	return out
}

func action(ctx *cli.Context) error {
	query := ctx.String("query")

	if len(strings.TrimSpace(query)) == 0 {
		// cli.ShowAppHelp(ctx)
		message := fmt.Sprintf("Query is not valid: '%s'", query)
		return cli.NewExitError(message, 1)
	}

	s, err := scraper.New(query)
	if err != nil {
		return cli.NewMultiError(err)
	}

	twChannel, err := s.Start()
	if err != nil {
		return cli.NewMultiError(err)
	}
	data := generalize(twChannel)

	// Check if we must enrich the tweets
	if len(ctx.String("key")) != 0 {
		data, err = configureTwitter(ctx, data)
		if err != nil {
			return cli.NewMultiError(err)
		}
	}

	// Check if db must be used
	if len(ctx.String("database")) != 0 {
		data, err = configureMongo(ctx, data)
		if err != nil {
			return cli.NewMultiError(err)
		}
	}

	for tweet := range data {
		log.Printf("Got tweet from channel: %s", tweet)
		// processTweet(tweet)
	}

	return nil
}

func main() {
	app := cli.NewApp()

	app.Name = "Twitter Scraper"
	app.Usage = "Scrape tweets from Twitter"
	app.Version = "1.0.0"

	app.Flags = []cli.Flag{
		// Query
		cli.StringFlag{
			Name:   "query, q",
			Usage:  "`QUERY` to use",
			EnvVar: "TW_QUERY",
		},

		// Twitter
		cli.StringFlag{
			Name:   "key, k",
			Usage:  "`KEY` to use",
			EnvVar: "TW_KEY",
		},
		cli.StringFlag{
			Name:   "secret, s",
			Usage:  "`SECRET` to use",
			EnvVar: "TW_SECRET",
		},
		cli.StringFlag{
			Name:   "token, t",
			Usage:  "`TOKEN` to use",
			EnvVar: "TW_TOKEN",
		},
		cli.StringFlag{
			Name:   "token-secret, y",
			Usage:  "`TOKEN SECRET` to use",
			EnvVar: "TW_TOKEN_SECRET",
		},

		// Mongo
		cli.IntFlag{
			Name:   "port, p",
			Value:  27017,
			Usage:  "`PORT` to use",
			EnvVar: "TW_DB_PORT",
		},
		cli.StringFlag{
			Name:   "host, m",
			Value:  "localhost",
			Usage:  "`HOST` to use",
			EnvVar: "TW_DB_HOST",
		},
		cli.StringFlag{
			Name:   "database, d",
			Usage:  "`DATABASE` to use",
			EnvVar: "TW_DB_NAME",
		},
		cli.StringFlag{
			Name:   "collection, c",
			Value:  "tweets",
			Usage:  "`COLLECTION` to use",
			EnvVar: "TW_DB_COLLECTION",
		},
	}

	app.Action = action

	app.Run(os.Args)
}
