# crawler


A simple single-domain web crawler written in Go

## Getting Started

````
go get github.com/Pinkerton/crawler/cmd/crawler
cd $GOPATH/bin
./crawler pinkerton.io
````


## Notes

This was a fun way to learn Go! I ran into some really interesting concurrency challenges 
that really shaped how I designed the crawler. Some of the questions I faced were - how do I know when two different 
asynchronous groups of tasks that create work for each other are finished? After I answered that, my next question
regarded how to do it reliably, since crawling complex website seemed to *occasionally* execute successfully, while a very
simple one would never exit. 

If there weren't enough tasks available, some workers would sit around and spam a monitoring task to say they were bored. 
That monitoring task patiently went through *every* message sent by up to 20 other tasks on their bored / busy status. 
Of course this filled the channel's buffer, which caused the worker tasks to block when trying to queue up more messages. I resolved these issues two ways:

1. **Debouncing** - Just because all of the worker threads may be bored for a split second doesn't mean they're completely done executing. This usually happened *right* after requesting and indexing the first page of a website, presumably after the only active index worker sent messages on the channel and before a requesting thread could pick them up. It was tempting to throw a time.Sleep()
 in there and call it a day, but that's bush league. I'd used debouncing before when writing code to interact with hardware, and it seemed like the right tool here to guarantee the threads are *actually* done. Go seems very practically designed, so it wouldn't surprise me if it had built-in support for something similar.
 but my use
2. **Don't send unnecessary messages** - Instead, explicitly send a first message to the monitoring function so it doesn't kill the
task right away and then only post new messages on state changes.

[pprof](https://golang.org/pkg/net/http/pprof/) was very helpful in debugging some of the issues I ran into.

### Design Decisions

I debated between an approach where every crawler goroutine performs the entire crawling process for each page,
and where the crawling process is split into separate requesting and indexing goroutines. I went with the latter and feel my solution
is over-engineered, but is also more flexible if one wanted to carve out more pieces of the crawler into their own worker pools.
It would be worth comparing the performance of an "every task is a crawler" version to this implementation.

## Known Issues

 * The crawler thinks example.com and example.com/ are different pages.
 * Lacks fancy output formatting.
 * No command line arguments to control number of spawned goroutines.
 * No tests :(
 * No rate limiting
 * Not Google


A real search engine should play nicely with the sites it crawls, so I imagine they have a per-site rate limit of 
1-2 seconds. Threads / goroutines could crawl other sites while waiting on this per-site timer to expire and use a 
priority queue of the next links to fetch, sorted by the the shortest time remaining to hit that specific host again.

## Pseudocode

````
Get home url
Fetch the page at the url
Parse that page for its links
Put the page url, fetched page, and links into Webpage data structure
For each link:
    Check if url already in Webpage map, if not...
        Spawn / grab a thread out of a pool
        Make a request to the url
        Parse links
        Add to Webpage map
        Repeat
    If url is in map,
        Don't do anything
````
