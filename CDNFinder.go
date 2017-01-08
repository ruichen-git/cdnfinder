package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

type cdnVendor struct {
	Symbol, Vendor string
}

var maxURL = 100 //Max number of Url to check per page

var cdnVendors = []cdnVendor{
	{"cloudfront", "AWS CloudFront"},
	{"kunlun", "Alibaba Cloud"}, {"tbcache", "Alibaba Cloud"}, {"alicdn", "Alibaba Cloud"},
	{"ccgslb", "ChinaCache"}, {"lxdns", "ChinaCache"}, {"chinacache", "ChinaCache"},
	{"edgekey", "Akamai"}, {"akamai", "Akamai"},
	{"fastly", "Fastly"},
	{"edgecast", "EdgeCast"},
	{"azioncdn", "Azion"},
	{"cachefly", "CacheFly"},
	{"cdn77", "CDN77"},
	{"cdnetworks", "CDNetworks"},
	{"cdnify", "CDNify"},
	{"wscloudcdn", "ChinaNetCenter"}, {"speedcdns", "ChinaNetCenter/Quantil"}, {"mwcloudcdn", "ChinaNetCenter/Quantil"},
	{"cloudflare", "CloudFlare"},
	{"hwcdn", "HighWinds"},
	{"kxcdn", "KeyCDN"}, {"awsdns", "KeyCDN"},
	{"fpbns", "Level3"}, {"footprint", "Level3"},
	{"llnwd", "LimeLight"},
	{"netdna", "MaxCDN"},
	{"bitgravity", "Tata CDN"},
	{"azureedge", "Azure CDN"},
	{"anankecdn", "Anake CDN"},
	{"presscdn", "Press CDN"},
	{"telefonica", "Telefonica CDN"},
	{"dnsv1", "Tecent CDN"}, {"cdntip", "Tecent CDN"},
	{"skyparkcdn", "Sky Park CDN"},
	{"ngenix", "Ngenix"},
	{"lswcdn", "LeaseWeb"},
	{"internapcdn", "Internap"},
	{"incapdns", "Incapsula"},
	{"cdnsun", "CDN SUN"},
	{"cdnvideo", "CDN Video"},
}

var dMode = false
var wg sync.WaitGroup
var fmtMux sync.Mutex

func main() {
	var searchUrls []string

	urlPtr := flag.String("url", "", "Start Url")
	filePtr := flag.String("file", "", "Url File")
	debugPtr := flag.Bool("debug", false, "Debug Mode")

	flag.Parse()

	if *urlPtr == "" && *filePtr == "" {
		fmt.Println("Please specify either start url or url file")
		os.Exit(1)
	}

	if *urlPtr != "" && *filePtr != "" {
		fmt.Println("Please specify either start url or url file")
		os.Exit(2)
	}

	dMode = *debugPtr

	if *urlPtr != "" {
		searchUrls = append(searchUrls, *urlPtr)
	} else {
		fileReader, err := os.Open(*filePtr)

		if err != nil {
			fmt.Println("Fail to open file " + *filePtr)
			os.Exit(3)
		}

		lineReader := bufio.NewReader(fileReader)
		aLine, _, rErr := lineReader.ReadLine()

		for rErr == nil {
			searchUrls = append(searchUrls, string(aLine))
			aLine, _, rErr = lineReader.ReadLine()
		}

		fileReader.Close()
	}

	for _, startUrl := range searchUrls {

		wg.Add(1)
		insideLinks := crawlURL(startUrl)
		wg.Wait()

		wg.Add(len(insideLinks))
		for _, v := range insideLinks {
			go crawlURL(v)
		}
		wg.Wait()
	}

}

func findCDNVendor(domain string) string {
	for _, v := range cdnVendors {
		if strings.Contains(strings.ToLower(domain), v.Symbol) {
			return v.Vendor
		}
	}
	return ""
}

func crawlURL(urlStr string) []string {
	defer wg.Done()

	urlStr = strings.ToLower(urlStr)

	if !strings.HasPrefix(urlStr, "http") {
		urlStr = "http://" + urlStr
	}

	var err error
	var resp *http.Response
	resp, err = http.Get(urlStr)

	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	domainMap := make(map[string]string)
	cdnMap := make(map[string]string)

	trLinks := getLinks(resp.Body)

	links := append(trLinks, urlStr)

	for _, link := range links {

		if strings.HasPrefix(link, "//") == true {
			link = link[2:]
		}
		urlStr2, err := url.Parse(link)

		if err == nil {
			if _, ok := domainMap[urlStr2.Host]; !ok {
				cname, _ := net.LookupCNAME(urlStr2.Host)
				domainMap[urlStr2.Host] = cname
				cdn := findCDNVendor(cname)
				cdnMap[urlStr2.Host] = cdn
			}
		}

	}

	fmtMux.Lock()
	fmt.Println("Inspecting " + urlStr)
	fmt.Println("==========================================================================================================")
	for i, v := range domainMap {
		if cdnMap[i] != "" || dMode == true {
			fmt.Printf("%s\t\t%s\t\t%s\n", i, v, cdnMap[i])
		}
	}
	fmt.Println("")
	fmtMux.Unlock()

	//Filter out unnecessary urls for next round
	nrLinks := make([]string, len(trLinks))
	nrI := 0
	for _, v := range trLinks {
		if strings.HasSuffix(v, "htm") || strings.HasSuffix(v, "html") || strings.HasSuffix(v, "/") {
			nrLinks[nrI] = v
			nrI++
			continue
		}

		if (strings.LastIndex(v, ".") != len(v)-3) && (strings.LastIndex(v, ".") != len(v)-4) && (strings.LastIndex(v, ".") != len(v)-5) {
			nrLinks[nrI] = v
			nrI++
			continue
		}

	}

	return nrLinks[:nrI]
}

func getLinks(htmlBody io.ReadCloser) []string {
	thisRound := make([]string, maxURL)

	thisLen := 0

	z := html.NewTokenizer(htmlBody)

	for {
		tt := z.Next()

		switch {
		case tt == html.ErrorToken:

			return thisRound[:thisLen]

		case tt == html.StartTagToken:
			t := z.Token()
			urls := getAttrUrls(t.Attr)
			if urls != nil {
				for _, v := range urls {

					thisRound[thisLen] = v
					thisLen++

					if thisLen == maxURL-1 {
						return thisRound
					}
				}
			}
		}
	}

}

func getAttrUrls(Attr []html.Attribute) []string {

	if Attr != nil {
		urls := make([]string, len(Attr))

		i := 0

		for _, v := range Attr {
			url := strings.Trim(v.Val, " ")

			if strings.Index(url, "http") == 0 {

				urls[i] = url
				i++
				continue
			}

			if strings.Index(url, "//") == 0 {

				urls[i] = url
				i++
				continue
			}

		}

		return urls[:i]
	}

	return nil
}
