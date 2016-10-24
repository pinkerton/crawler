package main

import (
"fmt"
"os"

"crawler"
)

func PrintStaticAssets(site *crawler.Website) {
    fmt.Println("Static assets for %s", site.Domain)
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

    site := crawler.Crawler(os.Args[1])
    PrintStaticAssets(site)
}
