package crawler

import(
    "net/http"
    "net/url"

    "golang.org/x/net/html"
    "golang.org/x/net/html/atom"
)

// Get the value for a generic attribute key
func GetAttr(t html.Token, key string) string {
    for _, a := range t.Attr {
        if a.Key == key {
            return a.Val
        }
    }
    return "" // attr not found
}

// Get an absolute URL from a specific attribute key
// we have a static resource URL string that *may* be absolute OR relative
// let's normalize it to always be absolute and a url.URL instance
func GetAttrURL(host *url.URL, t html.Token, key string) *url.URL {
    val := GetAttr(t, key)

    link, err := url.Parse(val)
    if err != nil {
        panic(err)
    }
    
    RelToAbsURL(host, link)
    FixScheme(link)
    return link
}

func RelToAbsURL(host *url.URL, link *url.URL) {
    if !link.IsAbs() {
        link.Host = host.Host
    }
}

func FixScheme(link *url.URL) {
    if link.Scheme == "" {
        link.Scheme = "http"
    }
}

func SameHost(host *url.URL, link *url.URL) bool {
    return host.Host == link.Host
}

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
                    href := GetAttrURL(host, t, "href")
                    if SameHost(host, href) && len(href.String()) > 0 {
                        FixScheme(href)
                        links = append(links, *href)
                    }
                // Images: <img>, Javascript: <script>
                case atom.Img, atom.Script:
                    src := GetAttrURL(host, t, "src")
                    if SameHost(host, src) {
                        assets = append(assets, src.String())
                    }
                // CSS: <link>
                case atom.Link:
                    href := GetAttrURL(host, t, "href")
                    if SameHost(host, href) {
                        assets = append(assets, href.String())
                    }
                }
            }
        }
    return links, assets
}

