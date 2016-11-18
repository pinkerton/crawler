/*
Web crawler that generates a sitemap of all pages on the same host.
Uses multiple workers for requesting and "indexing" pages, although each worker
does slightly more than its name gives it credit for. Request workers fetch pages
and parse out their links and static assets. Index workers add pages to the sitemap
and figure out which links the request workers should crawl next.
*/
package crawler

import (
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"crawler/backfill"
)

const (
	NumWorkers        = 10
	TotalWorkers      = NumWorkers * 2
	MsgsBufferSize    = TotalWorkers * 8
	RequestBufferSize = 200
	IndexBufferSize   = 400
	DebounceTimeout   = 2 * time.Second
)

// Website represents a single website to scrape. All Pages should be on the same
// domain and multithreaded Page access is encouraged with the included mutex.
type Website struct {
	Domain url.URL
	Pages  map[string]Webpage
	Lock   sync.Mutex
}

// Webpage represents specific page on a website that we can identify with its URL.
// Has Links and static Assets that we care about scraping.
type Webpage struct {
	URL    url.URL
	Links  []url.URL
	Assets []string
}

// WorkerMsg is sent on a channel from crawler goroutines to a monitoring function
// to notify if the worker is busy or free.
type WorkerMsg struct {
	ID   int
	Busy bool
}

// CrawlerState holds state shared by worker goroutines.
type CrawlerState struct {
	WG    *sync.WaitGroup
	Links chan url.URL
	Pages chan Webpage
	Msgs  chan WorkerMsg
	Done  chan bool
}

// Crawler sets up channels and crawling goroutines. Blocks on a shared WaitGroup
// for everything to finish before cleaning up and returning the crawled site.
func Crawler(link url.URL) *Website {
	site := Website{
		Domain: link,
		Pages:  make(map[string]Webpage)}

	state := CrawlerState{
		WG:    &sync.WaitGroup{},
		Links: make(chan url.URL, RequestBufferSize),
		Pages: make(chan Webpage, IndexBufferSize),
		Msgs:  make(chan WorkerMsg, MsgsBufferSize),
		Done:  make(chan bool, TotalWorkers)}
	state.Links <- link

	// Convoluted way to create NumWorkers number of both Request and Index
	// worker goroutines while giving each a unique ID.
	for i := 0; i < NumWorkers*2; i += 2 {
		state.WG.Add(2)
		go RequestWorker(i, &state)
		go IndexWorker(i+1, &state, &site)
	}

	go MonitorCrawler(&state)
	state.WG.Wait()

	defer close(state.Pages)
	defer close(state.Links)
	defer close(state.Msgs)
	return &site
}

// MonitorCrawler listens for messages from other workers about their current status (busy/free).
// If all the workers are without work for a specific time interval, puts messages
// on a channel to instruct them to terminate. Debouncing the status messages from
// workers is important because there are conditions, specifically after crawling and
// indexing the root of the "site tree", where all workers are free for a moment.
// You should only need ONE MonitorCrawler goroutine.
func MonitorCrawler(state *CrawlerState) {
	workers := make(map[int]bool)
	all_free := false
	var timestamp time.Time

Loop:
	for {
		select {
		case msg := <-state.Msgs:
			workers[msg.ID] = msg.Busy
		default:
			if len(workers) == TotalWorkers && backfill.DeepCompare(workers, false) {
				// Debounce the "free" messages before terminating workers.
				if all_free && time.Since(timestamp) >= DebounceTimeout {
					// Terminate the workers.
					for i := 0; i < len(workers); i++ {
						state.Done <- true
					}

					close(state.Done)
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

// RequestWorker awaits URLS of pages to crawl on the links channel. Should be run as a
// goroutine, and multiple workers can run concurrently. After fetching a page,
// it parses out links and static assets on the page and sends them on a channel
// the IndexWorker. If there are no links available immediately on the channel,
// sends a message to the monitor that it has no work to do. The worker will
// continue doing this until it either finds more work to do or it receives a
// message from the monitor to terminate, in which case it will stop looping
// and decrement its WaitGroup counter.
func RequestWorker(id int, state *CrawlerState) {
	msg := WorkerMsg{id, true}
	first := true

Loop:
	for {
		select {
		case link := <-state.Links:
			// Tell the monitor we have work to do if our last msg was different.
			if !msg.Busy || first {
				msg.Busy = true
				first = false
				state.Msgs <- msg
			}

			response, err := http.Get(link.String())
			if err != nil {
				log.Printf("[%d] request failed for URL: %s\n", id, link.String())
				continue
			}
			log.Printf("[%d] requested %s\n", id, link.String())

			links, assets := backfill.ParseAssets(response)
			page := Webpage{link, links, assets}
			state.Pages <- page
		default:
			select {
			case <-state.Done:
				break Loop
			default:
				if msg.Busy {
					msg.Busy = false
					state.Msgs <- msg
				}
			}
		}
	}
	state.WG.Done()
}

// IndexWorker awaits parsed webpages on the pages channel, adds them to the sitemap, and
// sends any uncrawled links from the page back to the RequestWorker via the
// links channel. Because there can be many goroutine instances of this worker,
// it uses a mutex to modify the sitemap. It uses the same technique as the RequestWorker
// to notify the MonitorWorker of its status and to know when to terminate.
func IndexWorker(id int, state *CrawlerState, site *Website) {
	msg := WorkerMsg{id, true}
	first := true
Loop:
	for {
		select {
		case page := <-state.Pages:
			// Tell the Monitor that we have work to do
			if !msg.Busy || first {
				msg.Busy = true
				first = false
				state.Msgs <- msg
			}
			// Add page to the sitemap
			site.Lock.Lock()
			site.Pages[page.URL.Path] = page
			site.Lock.Unlock()
			log.Printf("[%d] indexed %s\n", id, page.URL.String())

			// Check the links on the page to find out what to crawl next
			for _, link := range page.Links {
				// Throw out links from different hosts
				if !backfill.SameHost(&link, &site.Domain) {
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
					state.Links <- link
				}
			}
		default:
			select {
			case <-state.Done:
				break Loop
			default:
				// Tell the MonitorWorker that we currently have no work to do
				if msg.Busy {
					msg.Busy = false
					state.Msgs <- msg
				}
			}
		}
	}
	state.WG.Done()
}
