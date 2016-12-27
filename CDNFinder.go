package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/jackdanger/collectlinks"
)

type cdnVendor struct {
	Symbol, Vendor string
}

var cdnVendors = []cdnVendor{
	{"cloudfront", "AWS CloudFront"},
	{"kunlun", "Alibaba Cloud"},
	{"ccgslb", "ChinaCache"},
	{"lxdns", "ChinaCache"},
	{"chinacache", "ChinaCache"},
	{"edgekey", "Akamai"},
	{"akamai", "Akamai"},
	{"fastly", "Fastly"},
	{"edgecast", "EdgeCast"},
	{"azioncdn", "Azion"},
	{"cachefly", "CacheFly"},
	{"cdn77", "CDN77"},
	{"cdnetworks", "CDNetworks"},
	{"cdnify", "CDNify"},
	{"wscloudcdn", "ChinaNetCenter"},
	{"cloudflare", "CloudFlare"},
	{"hwcdn", "HighWinds"},
	{"kxcdn", "KeyCDN"},
	{"awsdns", "KeyCDN"},
	{"fpbns", "Level3"},
	{"footprint", "Level3"},
	{"llnwd", "LimeLight"},
	{"netdna", "MaxCDN"},
	{"speedcdns", "ChinaNetCenter/Quantil"},
	{"mwcloudcdn", "ChinaNetCenter/Quantil"},
	{"bitgravity", "Tata CDN"},
	{"azureedge", "Azure CDN"},
	{"anankecdn", "Anake CDN"},
	{"presscdn", "Press CDN"},
	{"telefonica", "Telefonica CDN"},
	{"dnsv1", "Tecent CDN"},
	{"cdntip", "Tecent CDN"},
	{"skyparkcdn", "Sky Park CDN"},
	{"ngenix", "Ngenix"},
	{"lswcdn", "LeaseWeb"},
	{"internapcdn", "Internap"},
	{"incapdns", "Incapsula"},
	{"cdnsun", "CDN SUN"},
	{"cdnvideo", "CDN Video"},
}

var dMode = false

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		fmt.Println("Please specify start url")
		os.Exit(1)
	}

	if len(args) > 1 {

		if args[1] == "-d" {
			dMode = true
		}
	}

	secLinks := crawlOne(args[0])
	for _, v := range secLinks {
		crawlOne(v)
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

func crawlOne(urlStr string) []string {
	fmt.Println("Inspecting " + urlStr)
	fmt.Println("==========================================================================")

	urlStr = strings.ToLower(urlStr)

	if !strings.HasPrefix(urlStr, "http") {
		urlStr = "http://" + urlStr
	}

	var err error
	var resp *http.Response
	resp, err = http.Get(urlStr)

	/*
		if strings.HasSuffix(urlStr, "htm") || strings.HasSuffix(urlStr, "html") || strings.HasSuffix(urlStr, "/") {
			resp, err = http.Get(urlStr)
		} else {
			return nil
		}
	*/

	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	domainMap := make(map[string]string)

	innerLinks := collectlinks.All(resp.Body)

	links := append(innerLinks, urlStr)

	nextLinks := make([]string, len(innerLinks)+1)
	i := 0

	for _, link := range links {

		if (strings.HasPrefix(link, "/") == false) || (strings.HasPrefix(link, "//") == true) {
			urlStr2, err := url.Parse(link)

			nextLinks[i] = urlStr2.String()
			i++

			if err == nil {
				if _, ok := domainMap[urlStr2.Host]; !ok {
					cname, _ := net.LookupCNAME(urlStr2.Host)
					domainMap[urlStr2.Host] = cname
					cdn := findCDNVendor(cname)
					if cdn != "" || dMode == true {
						fmt.Printf("%s\t%s\tusing %s\n", urlStr2.Host, cname, cdn)
					}
				}
			}
		}
	}
	fmt.Println("")

	return nextLinks[:i]
}
