package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"crawler"
)

func PrintStaticAssets(site *crawler.Website) {
	fmt.Printf("%s:\n", site.Domain.String())
	for link, page := range site.Pages {
		fmt.Printf("\t%s\n", link)

		fmt.Printf("\tLINKS\n")
		if len(page.Links) > 0 {
			for _, link := range page.Links {
				fmt.Printf("\t\t%s\n", link.String())
			}
		} else {
			fmt.Printf("\t\tN/A (no external links found)\n")
		}

		fmt.Printf("\tASSETS\n")
		if len(page.Assets) > 0 {
			for _, asset := range page.Assets {
				fmt.Printf("\t\t%s\n", asset)
			}
		} else {
			fmt.Printf("\t\tN/A (assets may be inlined)\n")
		}
		fmt.Printf("\n")
	}
}

func main() {
	// Handle errors
	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
		}
	}()

	if len(os.Args) != 2 {
		fmt.Printf("Usage: ./%s [url]\n", os.Args[0])
		os.Exit(1)
	}

	link := os.Args[1]
	if !strings.HasPrefix(link, "http") {
		link = "http://" + link
	}

	u, err := url.Parse(link)
	if err != nil {
		fmt.Println("Error! Malformed URL.")
		os.Exit(2)
	}

	site := crawler.Crawler(*u)
	PrintStaticAssets(site)
}
