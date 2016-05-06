package scraper

import (
	"net/url"

	"net/http"

	"io/ioutil"

	"encoding/json"

	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/pkg/errors"
	"golang.org/x/net/html"
)

type jsonResponse struct {
	MinPosition string `json:"min_position"`
	HTML        string `json:"items_html"`
}

func twitterURL(query string, mp *maxPosition) *url.URL {

	twURL := url.URL{
		Scheme: "https",
		Host:   twHost,
	}

	qs := twURL.Query()
	qs.Add("q", query)
	qs.Add("f", "tweets")
	qs.Add("vertical", "news")
	qs.Add("include_entities", "0")
	qs.Add("src", "sprv")

	if mp == nil {
		twURL.Path = twQueryPath
	} else {
		qs.Add("max_position", mp.String())
		twURL.Path = twAjaxPath
	}
	twURL.RawQuery = qs.Encode()

	return &twURL
}

func getPageContent(pageURL *url.URL) ([]byte, error) {
	resp, err := http.Get(pageURL.String())
	if err != nil {
		return nil, errors.Wrap(err, "Cannot get page")
	}

	body, err := ioutil.ReadAll(resp.Body)
	return body, nil
}
func parseHTML(document string) (*goquery.Document, error) {
	stringReader := strings.NewReader(document)
	node, err := html.Parse(stringReader)
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot parse html")
	}

	doc := goquery.NewDocumentFromNode(node)
	return doc, nil
}
func getHTMLPage(pageURL *url.URL) (*goquery.Document, error) {
	body, err := getPageContent(pageURL)
	if err != nil {
		return nil, errors.Wrap(err, "Cannot get HTML page")
	}

	return parseHTML(string(body))
}
func getJSONPage(pageURL *url.URL) (*goquery.Document, string, error) {
	body, err := getPageContent(pageURL)
	if err != nil {
		return nil, "", errors.Wrap(err, "Cannot get JSON page")
	}

	response := new(jsonResponse)
	err = json.Unmarshal(body, response)
	if err != nil {
		return nil, "", errors.Wrap(err, "Cannot unmarshall json into response")
	}

	HTML := response.HTML
	minPosition := parseMaxPosition(response.MinPosition)

	doc, err := parseHTML(HTML)
	if err != nil {
		return nil, "", errors.Wrap(err, "Cannot parse JSON html")
	}

	return doc, minPosition.last, nil
}
