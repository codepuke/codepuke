package web

import (
	"encoding/xml"
	"net/http"
	"time"
)

type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Items       []rssItem `xml:"item"`
}

type rssItem struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	GUID    string `xml:"guid"`
	PubDate string `xml:"pubDate"`
}

func (h *handlers) rss(w http.ResponseWriter, r *http.Request) {
	articles, err := h.store.ListPublishedArticles(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}

	feed := rssFeed{
		Version: "2.0",
		Channel: rssChannel{
			Title:       "codepuke",
			Link:        h.baseURL + "/",
			Description: "dated records on the gob wire format and its ports",
		},
	}
	for _, a := range articles {
		link := h.baseURL + "/articles/" + a.Slug
		feed.Channel.Items = append(feed.Channel.Items, rssItem{
			Title:   a.Title,
			Link:    link,
			GUID:    link,
			PubDate: a.PublishedAt.UTC().Format(time.RFC1123Z),
		})
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Write([]byte(xml.Header))
	if err := xml.NewEncoder(w).Encode(feed); err != nil {
		return
	}
}
