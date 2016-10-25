package crawler

import (
    "fmt"
    "net/http"
    "log"
    "net/url"
    "sync"
    "time"
)


const (
    NUM_WORKERS = 5
)

type Website struct {
    Domain url.URL
    Pages  map[string]Webpage
    Lock    sync.Mutex
}

type Webpage struct {
    URL    url.URL      // url of this page
    Links  []url.URL    // pages this one links to
    Assets []string     // slice of static assets included on this page
}

func IndexWorker(wg *sync.WaitGroup, urls chan<- url.URL, pages <-chan Webpage, 
                done chan bool, once sync.Once, site *Website) {
    Loop:
        for {
            select {
            case page := <-pages:
                wg.Add(1)

                // add page to the sitemap
                fmt.Printf("Indexed: %s\n", page.URL.String())
                site.Lock.Lock()
                site.Pages[page.URL.String()] = page
                site.Lock.Unlock()

                // check the links on the page to find out what to crawl next
                for _, link := range page.Links {
                    site.Lock.Lock()
                    _, ok := site.Pages[link.String()]
                    if !ok {  // we have not already crawled this link
                        // create a blank placeholder value so mulitple threads
                        // don't waste time requesting the same url
                        site.Pages[link.String()] = Webpage{}
                    }
                    site.Lock.Unlock()
                    if !ok {
                        // avoid risk holding the map mutex for longer than we need
                        // in case the channel's buffer is full and it blocks
                        urls <- link 
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
}

func Crawler(link url.URL) *Website {
    site := Website{Domain: link}
    site.Pages = make(map[string]Webpage)
    pages := make(chan Webpage, 1000)
    urls := make(chan url.URL, 1000)
    done := make(chan bool, NUM_WORKERS)

    var wg sync.WaitGroup
    var once sync.Once
    urls <- link

    for i := 1; i <= NUM_WORKERS; i++ {
        go RequestWorker(&wg, urls, pages)
        go IndexWorker(&wg, urls, pages, done, once, &site)
    }

    time.Sleep(2 * time.Second)
    wg.Wait()
    fmt.Println("Done!")
    for i := 1; i <= NUM_WORKERS; i++ {
        done <- true
    }

    close(pages)
    close(urls)
    close(done)
    return &site
}

func RequestWorker(wg *sync.WaitGroup, urls <-chan url.URL, pages chan<- Webpage) {
    for link := range urls {
        wg.Add(1)

        response, err := http.Get(link.String())
        if err != nil {
            log.Println("Request failed for URL: ", link.String())
            wg.Done()
            continue
        }

        links, assets := ParseAssets(response)
        page := Webpage{link, links, assets}
        pages <- page
        //fmt.Printf("%s\n", link.String())
        wg.Done()
    }
}


