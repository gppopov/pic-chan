package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

func main() {
	//url := "https://8ch.net/pol/res/8407765.html"
	if len(os.Args) < 2 {
		fmt.Printf("\nURL missing.\n\n")
		fmt.Printf("Usage:\n > pic-chan http://8ch.net/pol/res/100000.html\n\n")
		return
	}

	url := os.Args[1]
	is8ch, _ := regexp.MatchString("8ch.net", url)
	if !is8ch {
		fmt.Println("Current version supports only 8chan threads.")
		return
	}
	resp, _ := http.Get(url)
	z := html.NewTokenizer(resp.Body)
	urls := make([]string, 0, 200)
	i := 0

OuterLoop:
	for {
		tt := z.Next()

		switch {
		case tt == html.StartTagToken:
			t := z.Token()
			isAnchor := t.Data == "a"
			if isAnchor {
				for _, a := range t.Attr {
					if a.Key == "href" {
						isMediaImg, _ := regexp.MatchString("file_store", a.Val)
						if isMediaImg {
							urls = append(urls, a.Val)
							i++
							break
						}
					}
				}
			}
		case tt == html.ErrorToken:
			resp.Body.Close()
			break OuterLoop
		}
	}

	results := asyncHTTPGets(urls)

	for _ = range urls {
		result := <-results
		fmt.Printf("%s status: %s\n", result.url, result.response.Status)
	}
}

func asyncHTTPGets(urls []string) <-chan *HTTPResponse {
	ch := make(chan *HTTPResponse, len(urls)) // buffered
	for _, url := range urls {
		go func(url string) {
			resp, err := http.Get(url)
			saveFromResponse(resp, url)
			if resp != nil {
				defer resp.Body.Close()
			}
			ch <- &HTTPResponse{url, resp, err}
		}(url)
	}
	return ch
}

func saveFromResponse(resp *http.Response, url string) {
	tokens := strings.Split(url, "/")
	fileName := tokens[len(tokens)-1]

	if _, err := os.Stat(fileName); err == nil {
		return
	}

	output, err := os.Create(fileName)
	if err != nil {
		fmt.Println("Error while creating", fileName, "-", err)
		return
	}
	defer output.Close()

	n, err := io.Copy(output, resp.Body)
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println(n, "bytes downloaded.")

}

// HTTPResponse type describes a simple http response wiht error.
type HTTPResponse struct {
	url      string
	response *http.Response
	err      error
}
