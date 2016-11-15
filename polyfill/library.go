package crawler_polyfill

import (
    "errors"
    "net/url"

    "golang.org/x/net/html"
)

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
