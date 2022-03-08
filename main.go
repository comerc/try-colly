package main

import (
	"fmt"
	"time"

	"github.com/gocolly/colly"
)

func main() {
	c := colly.NewCollector(
		// MaxDepth is 1, so only the links on the scraped page
		// is visited, and no further links are followed
		colly.MaxDepth(1),

		// Visit only domains: hackerspaces.org, wiki.hackerspaces.org
		colly.AllowedDomains("go-colly.org"),
	)
	c.SetRequestTimeout(120 * time.Second)

	// Find and visit all links
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		// e.Request.Visit(e.Attr("href"))

		// link := e.Attr("href")
		// // Print link
		// fmt.Printf("Link found: %q -> %s\n", e.Text, link)
		// // Visit link found on page
		// // Only those links are visited which are in AllowedDomains
		// c.Visit(e.Request.AbsoluteURL(link))

		link := e.Request.AbsoluteURL(e.Attr("href"))
		if link != "" {
			fmt.Printf("Link found: %q -> %s\n", e.Text, link)
			c.Visit(link)
		}
	})

	// c.OnRequest(func(r *colly.Request) {
	// 	fmt.Printf("Visiting %s\n", r.URL)
	// })

	// c.OnResponse(func(r *colly.Response) {

	// 	fmt.Printf("Visited ContentType: %s FileName: %s\n", r.Headers.Get("Content-Type"), r.FileName())
	// })

	// c.OnScraped(func(r *colly.Response) {})

	c.OnError(func(r *colly.Response, e error) {
		fmt.Println("Got this error:", e)
	})

	c.Visit("http://go-colly.org/")
}
