package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/publicsuffix"
)

var (
	urlFile = flag.String("f", "", "url file ")
	urlPath = flag.String("p", "", "url path ")
	sp      = flag.String("sp", "class", "save path")
	sep     = flag.String("sep", "#", "sep")
)

func main() {

	//fmt.Println(strings.Join([]string{u.Subdomain, u.Domain, u.TLD}, "."))
	//fmt.Println(u.String())
	flag.Parse()
	domains := map[string][]string{}
	var err error
	if *urlFile != "" {
		domains, err = ReadFromTxt(*urlFile)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
	}
	if *urlPath != "" {
		filepath.Walk(*urlPath, func(path string, info fs.FileInfo, err error) error {
			if !info.IsDir() {
				fmt.Println(path)
				domains, _ = ReadFromTxt(path, domains)
			}
			return nil
		})
	}

	if err := os.MkdirAll(*sp, 0755); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	af, err := os.OpenFile(*sp+"/all.txt", os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer af.Close()

	for k, v := range domains {
		f, _ := os.OpenFile(*sp+"/"+k+".txt", os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
		defer f.Close()
		f.Write([]byte(strings.Join(v, "\n")))
		af.Write([]byte("\n" + strings.Join(v, "\n")))
	}

}

// URL embeds net/url and adds extra fields ontop
type URL struct {
	Subdomain, Domain, TLD, Port string
	ICANN                        bool
	*url.URL
}

// ReadLines reads all lines of the file.
func ReadFromTxt(path string, dms ...map[string][]string) (map[string][]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	domains := map[string][]string{}
	if len(dms) > 0 {
		domains = dms[0]
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		v := scanner.Text()
		if v != "" {
			tmp := strings.Split(v, *sep)
			if len(tmp) > 0 && tmp[0] != "" {
				url, err := parse(tmp[0])
				if err == nil {
					domains[url.TLD] = append(domains[url.TLD], tmp[0])
				}
			}
		}
	}

	return domains, scanner.Err()
}

// Parse mirrors net/url.Parse except instead it returns
// a tld.URL, which contains extra fields.
func parse(s string) (*URL, error) {
	// if !strings.Contains(s, "http") {
	// 	s = "http://" + s
	// }
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
