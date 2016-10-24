package crawler

import (
    "fmt"
    "log"
    "net/url"
    "sync"
)

type Website struct {
    Domain url.URL
    Pages  map[url.URL]Webpage
    //Limiter *time.Ticker
}

type Webpage struct {
    URL    url.URL      // url of this page
    Links  []url.URL    // pages this one links to
    Assets []string     // slice of static assets included on this page
}

func WaitForDone(wg sync.WaitGroup, done chan bool) {
        // wait for the WaitGroup counter to be zero
        wg.Wait()
        fmt.Println("Done!")
        done <- true
}



func Crawler(link url.URL) *Website {
    site := Website{Domain: link}
    site.Pages = make(map[url.URL]Webpage)
    pages := make(chan Webpage, 500)
    urls := make(chan url.URL, 500)
    done := make(chan bool)
    var wg sync.WaitGroup
    first := true
    urls <- link

    for i := 1; i <= 5; i++ {
        go RequestWorker(wg, urls, pages)
    }

    Loop:
        for {
            select {
            case page := <-pages:
                wg.Add(1)
                fmt.Printf("Adding to sitemap: %s\n", page.URL.String())
                // add it to the sitemap
                site.Pages[page.URL] = page

                // check the links on the page to find out what to crawl next
                for _, link := range page.Links {
                    _, ok := site.Pages[link]
                    if !ok {    // we have not already crawled this link
                        fmt.Printf("To crawl: %s\n", link.String())
                        urls <- link
                    }

                    // make sure the pool has work to do before starting the 
                    // routine that will stop the crawler when the workers are done.
                    if first {
                        go WaitForDone(wg, done)
                        first = false
                    }
                }
                wg.Done()
            default:            // no page on the channel, check if we're done crawling
                select {
                case <-done:    // done crawling
                    fmt.Println("Done crawling!")
                    break Loop
                default:
                    continue    // workers must still be busy
                }
            }
        }

    close(pages)
    close(urls)
    close(done)
    //site.Limiter.Stop()   
    return &site
}

func RequestWorker(wg sync.WaitGroup, urls <-chan url.URL, pages chan<- Webpage) {
    defer func() {
        if err := recover(); err != nil {
            log.Println("SOSOSOSOS:", err)
            wg.Done()
            go RequestWorker(wg, urls, pages)
        }
    }()

    for link := range urls {
        wg.Add(1)

        //fmt.Printf("Requesting URL: %s\n", link.String())
        response, err := Fetch(link)
        if err != nil {
            wg.Done()
            continue
        }

        links, assets := ParseAssets(response)
        page := Webpage{link, links, assets}
        pages <- page
        fmt.Printf("Reqested URL: %s\n", link.String())
        wg.Done()
    }
}


