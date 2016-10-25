/*
Web crawler that generates a sitemap of all pages on the same host.
Uses multiple workers for requesting and "indexing" pages, although each worker
does slightly more than its name gives it credit for. Request workers fetch pages
and parse out their links and static assets. Index workers add pages to the sitemap
and figure out which links the request workers should crawl next.
*/
package crawler

import (
    "net/http"
    "log"
    "net/url"
    "sync"
    "time"
)


const (
    NumWorkers = 10
    TotalWorkers = NumWorkers * 2
    MsgsBufferSize = TotalWorkers * 8
    RequestBufferSize = 200
    IndexBufferSize = 400
    TwoSeconds = 3 * time.Second
)

// Represents a single website to scrape. All Pages should be on the same
// domain and multithreaded Page access is encouraged with the included mutex.
type Website struct {
    Domain url.URL
    Pages  map[string]Webpage
    Lock    sync.Mutex
}

// A specific page on a website that we can identify with its URL. 
// Has Links and static Assets that we care about scraping.
type Webpage struct {
    URL    url.URL
    Links  []url.URL
    Assets []string
}

// Message sent on a channel from crawler goroutines (RequestWorker and IndexWorker) 
// to a monitoring function (MonitorCrawler) to notify if the worker is busy or free.
type WorkerMsg struct {
    ID int
    Busy bool
}

// Sets up channels and crawling goroutines. Blocks on a shared WaitGroup
// for everything to finish before cleaning up and returning the crawled site.
func Crawler(link url.URL) *Website {
    site := Website{Domain: link}
    site.Pages = make(map[string]Webpage)
    pages := make(chan Webpage, IndexBufferSize)
    links := make(chan url.URL, RequestBufferSize)
    msgs := make(chan WorkerMsg, MsgsBufferSize)
    done := make(chan bool, TotalWorkers)
    var wg sync.WaitGroup
    links <- link

    // Convoluted way to create NumWorkers number of both Request and Index
    // worker goroutines while giving each a unique ID.
    for i := 0; i < NumWorkers * 2; i+=2 {
        wg.Add(2)
        go RequestWorker(i, &wg, links, pages, msgs, done)
        go IndexWorker(i+1, &wg, links, pages, msgs, done, &site)
    }

    go MonitorCrawler(msgs, done)
    wg.Wait()

    defer close(pages)
    defer close(links)
    defer close(msgs)
    return &site
}

/* Listen for messages from other workers about their current status (busy/free).
   If all the workers are without work for a specific time interval, puts messages
   on a channel to instruct them to terminate. Debouncing the status messages from
   workers is important because there are conditions, specifically after crawling and 
   indexing the root of the "site tree", where all workers are free for a moment.
   You should only need ONE MonitorCrawler goroutine. */
func MonitorCrawler(msgs chan WorkerMsg, done chan<- bool) {
    workers := make(map[int]bool)
    all_free := false
    var timestamp time.Time

    Loop:
        for {
            select {
            case msg := <-msgs:
                workers[msg.ID] = msg.Busy
            default:
                if len(workers) == TotalWorkers && AllValuesEqual(workers, false) {
                    // Debounce the "free" messages before terminating workers.
                    if all_free && time.Since(timestamp) >= TwoSeconds {
                        // Terminate the workers.
                        for i := 0; i < len(workers); i++ {
                            done <- true
                        }
                        
                        close(done)
                        break Loop
                    } else if !all_free {
                        // Workers are free for at least this moment, start timer.
                        all_free = true
                        timestamp = time.Now()
                    }
                } else {
                    // A worker became busy, reset.
                    all_free = false
                }
            }
    }
}

/* Awaits URLS of pages to crawl on the links channel. Should be run as a
   goroutine, and multiple workers can run concurrently. After fetching a page,
   it parses out links and static assets on the page and sends them on a channel 
   the IndexWorker. If there are no links available immediately on the channel, 
   sends a message to the monitor that it has no work to do. The worker will 
   continue doing this until it either finds more work to do or it receives a 
   message from the monitor to terminate, in which case it will stop looping 
   and decrement its WaitGroup counter.
*/
func RequestWorker(id int, wg *sync.WaitGroup, links <-chan url.URL, pages chan<- Webpage, 
                msgs chan WorkerMsg, done <-chan bool) {
    msg := WorkerMsg{id, true}
    first := true

    Loop:
        for {
            select {
            case link := <-links:
                // Tell the monitor we have work to do if our last msg was different.
                if !msg.Busy || first {
                    msg.Busy = true
                    first = false
                    msgs <- msg
                }
                
                response, err := http.Get(link.String())
                if err != nil {
                    log.Printf("[%d] request failed for URL: %s\n", id, link.String())
                    continue
                }
                log.Printf("[%d] requested %s\n", id, link.String())

                links, assets := ParseAssets(response)
                page := Webpage{link, links, assets}
                pages <- page
            default:
                select {
                case <-done:
                    break Loop
                default:
                    if msg.Busy {
                        msg.Busy = false
                        msgs <- msg
                    }
                }
            }
        }
    wg.Done()
}

/* Awaits parsed webpages on the pages channel, adds them to the sitemap, and
   sends any uncrawled links from the page back to the RequestWorker via the 
   links channel. Because there can be many goroutine instances of this worker,
   it uses a mutex to modify the sitemap. It uses the same technique as the RequestWorker
   to notify the MonitorWorker of its status and to know when to terminate.
*/
func IndexWorker(id int, wg *sync.WaitGroup, links chan<- url.URL, pages <-chan Webpage, 
                msgs chan WorkerMsg, done <-chan bool, site *Website) {
    msg := WorkerMsg{id, true}
    first := true
    Loop:
        for {
            select {
            case page := <-pages:
                // Tell the Monitor that we have work to do
                if !msg.Busy || first {
                    msg.Busy = true
                    first = false
                    msgs <- msg
                }
                // Add page to the sitemap
                site.Lock.Lock()
                site.Pages[page.URL.Path] = page
                site.Lock.Unlock()
                log.Printf("[%d] indexed %s\n", id, page.URL.String())

                // Check the links on the page to find out what to crawl next
                for _, link := range page.Links {
                    // Throw out links from different hosts
                    if !SameHost(&link, &site.Domain) {
                        continue
                    }

                    site.Lock.Lock()
                    _, ok := site.Pages[link.Path]
                    if !ok {  
                        // We have not already crawled this URL; create a placeholder
                        // so mulitple threads don't end up requesting the same link.
                        site.Pages[link.Path] = Webpage{}
                    }
                    site.Lock.Unlock()

                    // Avoid the risk holding the mutex while putting something
                    // on the links channel in case its buffer is full. This would
                    // would block ALL IndexWorkers from using the sitemap
                    // until the channel's buffer had space and could cause deadlock.
                    if !ok {
                        links <- link 
                    }
                }
            default:
                select {
                    case <-done:
                    break Loop
                default:
                    // Tell the MonitorWorker that we currently have no work to do
                    if msg.Busy {
                        msg.Busy = false
                        msgs <- msg
                    }
                }
            }
        }
    wg.Done()
}
