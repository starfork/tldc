package main

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"

	"golang.org/x/net/publicsuffix"
)

func main() {
	dms, err := readlines("./url.txt")
	if err != nil {
		panic(err)
	}
	domains := map[string][]string{}
	for _, v := range dms {
		url, err := parse(v)
		fmt.Println(err)
		if err == nil {
			domains[url.TLD] = append(domains[url.TLD], v)
		}
	}

	for k, v := range domains {
		f, _ := os.OpenFile(k+".txt", os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
		defer f.Close()
		f.Write([]byte(strings.Join(v, "\n")))
	}

}

// URL embeds net/url and adds extra fields ontop
type URL struct {
	Subdomain, Domain, TLD, Port string
	ICANN                        bool
	*url.URL
}

// ReadLines reads all lines of the file.
func readlines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, scanner.Err()
}

// Parse mirrors net/url.Parse except instead it returns
// a tld.URL, which contains extra fields.
func parse(s string) (*URL, error) {
	url, err := url.Parse(s)
	if err != nil {
		return nil, err
	}
	if url.Host == "" {
		return &URL{URL: url}, nil
	}
	dom, port := domainPort(url.Host)
	//etld+1
	etld1, err := publicsuffix.EffectiveTLDPlusOne(dom)
	suffix, icann := publicsuffix.PublicSuffix(strings.ToLower(dom))
	// HACK: attempt to support valid domains which are not registered with ICAN
	if err != nil && !icann && suffix == dom {
		etld1 = dom
		err = nil
	}
	if err != nil {
		return nil, err
	}
	//convert to domain name, and tld
	i := strings.Index(etld1, ".")
	if i < 0 {
		return nil, fmt.Errorf("tld: failed parsing %q", s)
	}
	domName := etld1[0:i]
	tld := etld1[i+1:]
	//and subdomain
	sub := ""
	if rest := strings.TrimSuffix(dom, "."+etld1); rest != dom {
		sub = rest
	}
	return &URL{
		Subdomain: sub,
		Domain:    domName,
		TLD:       tld,
		Port:      port,
		ICANN:     icann,
		URL:       url,
	}, nil
}

func domainPort(host string) (string, string) {
	for i := len(host) - 1; i >= 0; i-- {
		if host[i] == ':' {
			return host[:i], host[i+1:]
		} else if host[i] < '0' || host[i] > '9' {
			return host, ""
		}
	}
	//will only land here if the string is all digits,
	//net/url should prevent that from happening
	return host, ""
}
