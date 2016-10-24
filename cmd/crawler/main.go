package main

import (
"fmt"
"net/url"
"os"

"crawler"
)

func PrintStaticAssets(site *crawler.Website) {
    fmt.Printf("Static assets for %s:\n", site.Domain)
    for url, page := range site.Pages {
        fmt.Println("\t%s", url)
        for _, asset := range page.Assets {
            fmt.Println("\t\t%s", asset)
        }
    }
}

func main() {
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
