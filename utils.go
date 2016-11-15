package crawler

import (
	"errors"
	"net/http"
	"net/url"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Returns true if all values in a map[int][bool] are equal to the passed value
func AllValuesEqual(items map[int]bool, value bool) bool {
	for _, flag := range items {
		if flag != value {
			return false
		}
	}
	return true
}

// Get the value for an attribute key in an HTML open tag.
// For example, in <a href="foo">, key = "href", val = "foo".
func GetAttr(t html.Token, key string) (string, error) {
	for _, a := range t.Attr {
		if a.Key == key {
			return a.Val, nil
		}
	}
	err := errors.New("key not found")
	return "", err // attr not found
}

// Get an absolute URL from a specific attribute key.
func GetAttrURL(host *url.URL, t html.Token, key string) (link *url.URL, err error) {
	val, err := GetAttr(t, key)
	if err != nil {
		return link, err
	}

	link, err = url.Parse(val)
	if err != nil {
		panic(err)
	}

	RelToAbsURL(host, link)
	FixScheme(link)
	return link, err
}

// Get an absolute URL from a relative one.
func RelToAbsURL(host *url.URL, link *url.URL) {
	if !link.IsAbs() {
		link.Host = host.Host
	}
}

// Add default HTTP scheme to URLs without it.
func FixScheme(link *url.URL) {
	if link.Scheme == "" {
		link.Scheme = "http"
	}
}

// Determines if two URLs share the same host.
func SameHost(u *url.URL, v *url.URL) bool {
	return u.Host == v.Host
}

// Parses links and static assets out of an HTML document.
func ParseAssets(response *http.Response) (links []url.URL, assets []string) {
	host := response.Request.URL

	z := html.NewTokenizer(response.Body)
	defer response.Body.Close()

Loop:
	for {
		tt := z.Next()
		switch {
		case tt == html.ErrorToken:
			// Done parsing the document
			break Loop
		case tt == html.StartTagToken:
			t := z.Token()
			switch t.DataAtom {
			// Links: <a>
			case atom.A:
				href, err := GetAttrURL(host, t, "href")
				if err == nil && SameHost(host, href) && len(href.String()) > 0 {
					FixScheme(href)
					links = append(links, *href)
				}
			// Images: <img>, Javascript: <script>
			case atom.Img, atom.Script:
				src, err := GetAttrURL(host, t, "src")
				if err == nil && SameHost(host, src) {
					assets = append(assets, src.String())
				}
			// CSS: <link>
			case atom.Link:
				href, err := GetAttrURL(host, t, "href")
				if err == nil && SameHost(host, href) {
					assets = append(assets, href.String())
				}
			}
		}
	}
	return links, assets
}
