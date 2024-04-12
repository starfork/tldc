package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/publicsuffix"

	"github.com/xuri/excelize/v2"
)

var (
	urlFile = flag.String("f", "url.xlsx", "url file ")
	sheet   = flag.String("sheet", "Sheet1", "sheet name")
	sp      = flag.String("sp", "筛选", "save path")
)

func main() {

	flag.Parse()
	var domains map[string][]string
	var err error
	ext := filepath.Ext(*urlFile)
	if ext == ".xlsx" {
		domains, err = ReadFromExcel(*urlFile)
	} else if ext == ".txt" {
		domains, err = ReadFromTxt(*urlFile)
	} else {
		panic("unsupport file")
	}

	if err != nil {
		panic(err)
	}

	for k, v := range domains {
		f, _ := os.OpenFile(*sp+"/"+k+".txt", os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
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

func ReadFromExcel(path string) (map[string][]string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	rows, err := f.GetRows(*sheet)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	domains := map[string][]string{}
	for _, row := range rows {
		if row[0] != "" {
			url, err := parse(row[0])
			if err == nil {
				domains[url.TLD] = append(domains[url.TLD], row[0])
			}
		}
	}
	return domains, nil
}

// ReadLines reads all lines of the file.
func ReadFromTxt(path string) (map[string][]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	domains := map[string][]string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		v := scanner.Text()
		if v != "" {
			url, err := parse(v)
			if err == nil {
				domains[url.TLD] = append(domains[url.TLD], v)
			}
		}
	}

	return domains, scanner.Err()
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
