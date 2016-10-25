package main

import (
"fmt"
"net/url"
"os"

"crawler"
)

import _ "net/http/pprof"
import "log"
import "net/http"


func PrintStaticAssets(site *crawler.Website) {
    fmt.Printf("Static assets for %s:\n", site.Domain.String())
    for link, page := range site.Pages {
        fmt.Println("\t%s", link)
        for _, asset := range page.Assets {
            fmt.Println("\t\t%s", asset)
        }
    }
}

func main() {
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()

    if len(os.Args) != 2 {
        fmt.Println("Usage: ./crawler_cmd [url]")
        os.Exit(1)
    }

    u, err := url.Parse(os.Args[1])
    if err != nil {
        fmt.Println("Error! Malformed URL.")
        os.Exit(2)
    }
    site := crawler.Crawler(*u)
    PrintStaticAssets(site)
}
