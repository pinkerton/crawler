package backfill

import (
	"net/http"
	"net/url"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// ParseAssets parses links and static assets out of an HTML document.
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
