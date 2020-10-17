package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

type Vuln struct {
	Key   string `json:"exception"`
	Value string `json:"message"`
}

func main() {
	file, _ := os.Open("G:/result.txt")

	defer file.Close()

	reader := bufio.NewReader(file)
	for {
		var buffer bytes.Buffer

		var l []byte
		var isPrefix bool
		for {
			l, isPrefix, _ = reader.ReadLine()
			buffer.Write(l)

			if !isPrefix {
				break
			}
		}
		line := buffer.String()

		sc := bufio.NewScanner(strings.NewReader(line))

		urls := make(chan string, 128)
		concurrency := 12
		var wg sync.WaitGroup
		wg.Add(concurrency)

		for i := 0; i < concurrency; i++ {
			go func() {
				for raw := range urls {

					u, err := url.ParseRequestURI(raw)
					if err != nil {
						fmt.Printf("invalid url: %s\n", raw)
						continue
					}

					if !resolves(u) {
						fmt.Printf("does not resolve: %s\n", u)
						continue
					}

					resp, err := fetchURL(u)
					if err != nil {
						fmt.Printf("failed to fetch: %s (%s)\n", u, err)
						color.HiRed("[*] Not found URL exactly")
						continue
					}

					if resp.StatusCode != http.StatusOK {
						fmt.Printf("non-200 response code: %s (%s)\n", u, resp.Status)
						color.HiRed("[*] Can't Exploit")
					}

					if resp.StatusCode == http.StatusOK {
						fmt.Printf("200 response code: %s (%s)\n", u, resp.Status)
						buf := new(bytes.Buffer)
						buf.ReadFrom(resp.Body)
						newStr := buf.String()

						if checkFile(newStr) == true {
							color.HiGreen("[*] Vulnerable System Found!\n")

							f, err := os.OpenFile("text.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

							if err != nil {
								log.Println(err)
							}
							defer f.Close()
							if _, err := f.WriteString("" + u.String() + "/\n"); err != nil {
								log.Println(err)
							}
						}

						if checkFile(newStr) == false {
							color.HiRed("[*] Vulnerable System Not Found!\n")
						}
					}
				}
				wg.Done()
			}()
		}

		for sc.Scan() {
			urls <- sc.Text()
		}
		close(urls)

		if sc.Err() != nil {
			fmt.Printf("error: %s\n", sc.Err())
		}

		wg.Wait()

	}

}

func resolves(u *url.URL) bool {
	addrs, _ := net.LookupHost(u.Hostname())
	return len(addrs) != 0
}

func fetchURL(u *url.URL) (*http.Response, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := http.Client{
		Transport: tr,
		Timeout:   5 * time.Second,
	}

	req, err := http.NewRequest("POST", ""+u.String()+"/api/jsonws/invoke", nil)
	if err != nil {
		return nil, err
	}

	req.Close = true
	req.Header.Set("User-Agent", "liferay scanner/0.1")
	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	return resp, err
}

func checkFile(filename string) bool {
	var status bool
	status = false

	s := &Vuln{
		Key:   "java.lang.IllegalStateException",
		Value: "Unable to deserialize object",
	}

	s2, _ := json.Marshal(s)

	if strings.Contains(string(s2), filename) {
		status = true
	}

	return status
}
