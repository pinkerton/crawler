package crawler

import (
    "fmt"
)

type Website struct {
    Domain string
    Pages  map[string]Webpage
}

type Webpage struct {
    Url    string       // uri of this page
    Title  string       // <title> of the page
    Links  []string     // pages this one links to
    Assets []string     // slice of static assets included on this page
}


func Crawler(url string) *Website {
    site := Website{Domain: url}
    site.Pages = make(map[string]Webpage)
    urls := make(chan string, 10)
    pages := make(chan Webpage, 10)

    // start up a worker thread that will read the url from the channel
    urls <- url
    go RequestWorker(urls, pages)

    for page := range pages {
        // wait until we get a page on the channel
        fmt.Println("%s", page.Url)
        // add it to the sitemap
        site.Pages[page.Url] = page
        // iterate over every link
        for _, link := range page.Links {
            _, ok := site.Pages[link]
            if !ok {    // we have not already crawled this url
                urls <- link
            } 
        }

    }
    
    return &site
}

func RequestWorker(urls <-chan string, pages chan<- Webpage) {
    for url := range urls {
        response := Fetch(url)
        links := make([]string, 10)
        assets := make([]string, 10)
        title, links, assets := ParseAssets(response)
        page := Webpage{url, title, links, assets}
        pages <- page

    }
}
