package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
)

// Cases:
// + http://e38.ru/dir1/dir2/ // запишу index.html в папку /dir1/dir2/ // TODO: не работает
// + http://e38.ru/dir1/dir2/file // запишу file в папку /dir1/dir2/
// + http://e38.ru/shoutbox?page=50 // надо парсить каждый url
// + http://e38.ru/node/3474#comment-61752 // отсекаю до /node/3474
// + http://e38.ru/node/37019?scroll&all#comments // отсекаю до /node/37019
// + http://e38.ru/user/login?destination=node/3474%2523comment-form // игнорирую
// + http://e38.ru/user/register?destination=node/37113%2523comment-form // игнорирую
// + http://e38.ru/files/* // игнорирую

func main() {
	currentPath, err := os.Getwd()
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}

	{
		err := os.MkdirAll(path.Join(currentPath, "logs"), 0755)
		if err != nil {
			log.Printf("ERROR: %s", err)
		}
	}

	{
		err := os.MkdirAll(path.Join(currentPath, "visited"), 0755)
		if err != nil {
			log.Printf("ERROR: %s", err)
		}
	}

	argsWithProg := os.Args
	if len(argsWithProg) != 2 {
		fmt.Print("need domain")
		os.Exit(1)
	}
	domain := argsWithProg[1]

	log.SetFlags(log.LUTC | log.Ldate | log.Ltime | log.Lshortfile)
	logFile, err := os.OpenFile(path.Join(currentPath, "logs", domain+".log"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("error opening file: %v", err)
		os.Exit(1)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	visitedFilePath := path.Join(currentPath, "visited", domain+".txt")
	var visited []string
	if _, err := os.Stat(visitedFilePath); err == nil {
		data, err := os.ReadFile(visitedFilePath)
		if err != nil {
			log.Panicf("ERROR: %s", err)
		}
		visited = strings.Split(string(data), "\n")
	}
	visitedFile, err := os.OpenFile(visitedFilePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Printf("ERROR: %s", err)
	}
	defer visitedFile.Close()

	baseDir := path.Join(currentPath, "sites", domain)

	c := colly.NewCollector(
		colly.MaxDepth(1),
		colly.AllowedDomains(domain),
	)
	c.SetRequestTimeout(120 * time.Second)
	c.IgnoreRobotsTxt = false

	c.OnRequest(func(r *colly.Request) {
		log.Printf("Visit to %s\n", r.URL.Path)
	})

	c.OnResponse(func(r *colly.Response) {
		contentType := strings.Split(r.Headers.Get("Content-Type"), ";")[0]
		if contentType != "text/html" {
			return
		}
		urlPath := r.Request.URL.Path
		filePath := strings.Split(urlPath, "/")
		dirPath := strings.Join(filePath[:len(filePath)-1], "/")
		fileName := filePath[len(filePath)-1]
		urlQuery := r.Request.URL.RawQuery
		if urlQuery != "" && substr(urlQuery, 0, len("page")) == "page" {
			fileName = fileName + "?" + urlQuery
		}
		if fileName == "" {
			fileName = "index"
		}
		fileName = fileName + ".html"
		{
			err := os.MkdirAll(path.Join(baseDir, dirPath), 0755)
			if err != nil {
				log.Printf("ERROR: %s", err)
			}
		}
		{
			err := os.WriteFile(path.Join(baseDir, dirPath, fileName), r.Body, 0644)
			if err != nil {
				log.Printf("ERROR: %s", err)
			}
		}
		log.Printf("Saved to %s", path.Join(dirPath, fileName))
	})

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Request.AbsoluteURL(e.Attr("href"))
		URL, err := url.Parse(link)
		if err != nil {
			log.Printf("ERROR: %s", err)
		}
		urlPath := URL.Path
		if urlPath == "/user/login" || urlPath == "/user/register" || substr(urlPath, 0, len("/files/")) == "/files/" {
			return
		}
		shortLink := urlPath
		urlQuery := URL.RawQuery
		if urlQuery != "" && substr(urlQuery, 0, len("page")) == "page" {
			shortLink = shortLink + "?" + urlQuery
		}
		if indexOf(visited, shortLink) == -1 {
			_, err := visitedFile.WriteString(shortLink + "\n")
			if err != nil {
				log.Printf("ERROR: %s", err)
			}
			visitedFile.Sync()
			visited = append(visited, shortLink)
			c.Visit(link) // !!! link (а не shortLink)
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("ERROR: %s", err)
	})

	c.Visit(fmt.Sprintf("https://%s/", domain))
}

func indexOf(slice []string, item string) int {
	for i := range slice {
		if slice[i] == item {
			return i
		}
	}
	return -1
}

func substr(input string, start int, length int) string {
	asRunes := []rune(input)

	if start >= len(asRunes) {
		return ""
	}

	if start+length > len(asRunes) {
		length = len(asRunes) - start
	}

	return string(asRunes[start : start+length])
}
