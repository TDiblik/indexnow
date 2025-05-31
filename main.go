package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

var searchEngineToIndexNowMapping = map[string]string{
	"IndexNow":       "https://api.indexnow.org/indexnow",
	"Microsoft Bing": "https://www.bing.com/indexnow",
	"Naver":          "https://searchadvisor.naver.com/indexnow",
	"Seznam.cz":      "https://search.seznam.cz/indexnow",
	"Yandex":         "https://yandex.com/indexnow",
	"Yep":            "https://indexnow.yep.com/indexnow",
}

func main() {
	var keyArg string
	var sitemapUrlArg string

	flag.StringVar(&keyArg, "k", "", "Generated uuid key accesible from https://<domain>/<key>.txt")
	flag.StringVar(&sitemapUrlArg, "s", "", "Specify a core/index sitemap. Sitemaps referenced in this sitemap will get automatically takein into account as well.")

	flag.Parse()

	if keyArg == "" {
		log.Fatalln("You have to specify your key file, without it, it's impossible to index your sites. Please use the \"-k\" argument, or \"-h\" for help.")
	}
	if sitemapUrlArg == "" {
		log.Fatalln("You have to specify your core/index sitemap, without it, it's impossible to find urls to index. Please use the \"-s\" argument, or \"-h\" for help.")
	}

	sitemapUrl, err := url.Parse(sitemapUrlArg)
	if err != nil {
		log.Fatalln("Please specify a valid sitemap url. The url you specified caanot be parsed as valid url.")
	}

	if err := checkKeyFileIntegrity(keyArg, sitemapUrl); err != nil {
		log.Fatalln(err, "\nUnable to check the key file integritiy.")
	}
	log.Println("✅ Key file integrity check completed.")

	urlsToIndex, err := getUrlsFromSitemaps(sitemapUrl)
	if err != nil {
		log.Fatalln(err, "\nUnable to parse sitemap(s).xml properly.")
	}
	log.Println("✅ Got a list of sites to index: ", urlsToIndex)

	for searchEngineName, indexnowUrl := range searchEngineToIndexNowMapping {
		res, err := postToIndexNow(sitemapUrl, keyArg, urlsToIndex, indexnowUrl)
		if err != nil {
			log.Fatalln(err, "\nUnable submit to indexnow.")
		}
		switch res.StatusCode {
		case 200:
			log.Println("✅ URL(s) submitted successfully to ", searchEngineName, "!")
		case 202:
			log.Println("⚙️ URL(s) received. IndexNow key validation pending...", searchEngineName, "(response)")
		case 400:
			log.Println("❗️ Invalid format. Report this to: https://github.com/TDiblik/indexnow/issues/new.", searchEngineName, "(response)")
		case 403:
			log.Println("❗️ In case of key not valid (e.g. key not found, file found but key not in the file).", searchEngineName, "(response)")
		case 422:
			log.Println("❗️ In case of URLs which don't belong to the host or the key is not matching the schema in the protocol.", searchEngineName, "(response)")
		case 429:
			log.Println("❗️ Too Many Requests (potential Spam). ", searchEngineName, " (response)")
		}
	}

}

func checkKeyFileIntegrity(key string, siteUrl *url.URL) error {
	keyFileUrl := siteUrl.Scheme + "://" + siteUrl.Host + "/" + key + ".txt"
	res, err := http.Get(keyFileUrl)
	if err != nil || res.StatusCode != 200 {
		return fmt.Errorf("Unable to reach the site %s, \n error log: %v", keyFileUrl, err)
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("Unable to read the response body: %v", err)
	}

	resBodyString := strings.TrimSpace(string(resBody))
	if resBodyString != key {
		return fmt.Errorf("Unable to read the response body: %v", err)
	}

	return nil
}

type SitemapIndex struct {
	Sitemaps []SitemapEntry `xml:"sitemap"`
}

type SitemapEntry struct {
	Loc string `xml:"loc"`
}

type URLSet struct {
	URLs []URLEntry `xml:"url"`
}

type URLEntry struct {
	Loc string `xml:"loc"`
}

func getUrlsFromSitemaps(sitemapBaseUrl *url.URL) ([]string, error) {
	visited := make(map[string]bool)
	allURLs := []string{}

	var fetch func(string) error
	fetch = func(sitemapURL string) error {
		if visited[sitemapURL] {
			return nil
		}
		visited[sitemapURL] = true

		resp, err := http.Get(sitemapURL)
		if err != nil {
			return fmt.Errorf("failed to fetch %s: %v", sitemapURL, err)
		}
		defer resp.Body.Close()

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read body from %s: %v", sitemapURL, err)
		}

		// First try to parse as SitemapIndex
		var index SitemapIndex
		if err := xml.Unmarshal(data, &index); err == nil && len(index.Sitemaps) > 0 {
			for _, entry := range index.Sitemaps {
				if err := fetch(entry.Loc); err != nil {
					return err
				}
			}
			return nil
		}

		// Otherwise try to parse as URLSet
		var urlset URLSet
		if err := xml.Unmarshal(data, &urlset); err != nil {
			return fmt.Errorf("failed to parse XML from %s: %v", sitemapURL, err)
		}
		for _, url := range urlset.URLs {
			allURLs = append(allURLs, url.Loc)
		}
		return nil
	}

	err := fetch(sitemapBaseUrl.String())
	return allURLs, err
}

type IndexNowRequest struct {
	Host    string   `json:"host"`
	Key     string   `json:"key"`
	UrlList []string `json:"urlList"`
}

func postToIndexNow(siteUrl *url.URL, key string, urlsToIndex []string, indexnowUrl string) (resp *http.Response, err error) {
	request := IndexNowRequest{
		Host:    siteUrl.Host,
		Key:     key,
		UrlList: urlsToIndex,
	}
	jsonBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse into indexnow request body: %v", err)
	}

	return http.Post(indexnowUrl, "application/json; charset=utf-8", bytes.NewReader(jsonBody))
}
