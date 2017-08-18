package main

import (
	"crypto/md5"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const rateLimit = 9 // I read somewhere it's 9 requests per minute?

type server struct {
	reqCnt   int64
	accounts map[string]*account
	cache    map[string]*Chars
	lock     sync.Mutex
}

func newServer(accounts []*account) *server {
	accountMap := make(map[string]*account)
	for _, acct := range accounts {
		accountMap[acct.Encoded()] = acct
	}

	s := &server{
		accounts: accountMap,
		cache:    make(map[string]*Chars),
	}

	s.runReset()

	return s
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	guid := r.URL.Query().Get("guid")
	if strings.Contains(guid, "@") {
		guid = fmt.Sprintf("%x", md5.Sum([]byte(guid)))
	}

	acct, ok := s.accounts[guid]
	if !ok {
		w.WriteHeader(404)
		return
	}

	if r.URL.Path == "/account/verifyage" {
		w.Write([]byte("<Success/>")) // whatevs
		return
	}

	cacheKey := fmt.Sprintf("%s:%s", acct.Encoded(), r.URL.Path)

	if atomic.LoadInt64(&s.reqCnt) >= rateLimit {
		s.serveFromCache(w, cacheKey)
		return
	}

	// Assemble the rotmg request
	vals := url.Values{}
	for k, v := range r.URL.Query() {
		if k == "guid" || k == "password" {
			continue
		}
		for _, sv := range v {
			vals.Add(k, sv)
		}
	}
	vals.Set("guid", acct.User)
	vals.Set("password", acct.Password)
	vals.Set("muleDump", "true")

	rurl := "https://realmofthemadgod.appspot.com" + r.URL.Path + "?" + vals.Encode()

	req, err := http.NewRequest("GET", rurl, nil)
	if err != nil {
		w.WriteHeader(400)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		w.WriteHeader(resp.StatusCode)
		return
	}

	chars := Chars{}
	if err := xml.NewDecoder(resp.Body).Decode(&chars); err != nil {
		w.WriteHeader(500)
		return
	}

	if len(chars.NextCharID) == 0 {
		log.Printf("error - serving from cache")
		s.serveFromCache(w, cacheKey)
		return
	}

	s.lock.Lock()
	s.cache[cacheKey] = &chars
	s.lock.Unlock()

	log.Printf("serving %s", req.URL.Path)
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Wrapper{Query: Query{Results: Results{chars}}})
}

func (s *server) serveFromCache(w http.ResponseWriter, cacheKey string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	resp, ok := s.cache[cacheKey]
	if !ok {
		w.WriteHeader(404)
		return
	}

	xml.NewEncoder(w).Encode(resp)
}

func (s *server) runReset() {
	// Resets the request counter every minute
	go func() {
		c := time.Tick(time.Minute)
		for _ = range c {
			atomic.StoreInt64(&s.reqCnt, 0)
		}
	}()
}

type account struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

func (a *account) Encoded() string {
	return fmt.Sprintf("%x", md5.Sum([]byte(a.User)))
}

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("usage: muleproxy <config.json>\n")
		os.Exit(0)
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Printf("error opening config file: %s\n", err)
		os.Exit(1)
	}

	var accounts []*account
	if err := json.NewDecoder(f).Decode(&accounts); err != nil {
		fmt.Printf("error decoding config file: %s\n", err)
		os.Exit(1)
	}

	s := newServer(accounts)

	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("where am i? %s\n", err)
		os.Exit(1)
	}

	if _, err := os.Stat("muledump.html"); !os.IsNotExist(err) {
		// Running from the muledump directory, serve the files too
		http.Handle("/", http.FileServer(http.Dir(wd)))
	}

	http.Handle("/char/list", s)
	http.Handle("/account/verifyage", s)
	http.Handle("/config.json", http.NotFoundHandler())

	if err := http.ListenAndServe(":5353", nil); err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}

type Wrapper struct {
	Query Query `json:"query"`
}

type Query struct {
	Results Results `json:"results"`
}

type Results struct {
	Chars Chars `json:"Chars"`
}

type Chars struct {
	NextCharID  string `xml:"nextCharId,attr"`
	MaxNumChars string `xml:"maxNumChars,attr"`
	Char        []Char `xml:"Char"`
	Account     struct {
		Vault struct {
			Chest []string `xml:"Chest"`
		} `xml:"Vault"`
		Gifts string `xml:"Gifts"`
		Name  string `xml:"Name"`
		Stats struct {
			ClassStats struct {
				ObjectType string `xml:"objectType,attr"`
				BestLevel  string `xml:"BestLevel"`
				BestFame   string `xml:"BestFame"`
			}
			BestCharFame string `xml:"BestCharFame"`
			TotalFame    string `xml:"TotalFame"`
			Fame         string `xml:"Fame"`
		} `xml:"Stats"`
	} `xml:"Account"`
}

type Char struct {
	ID               string `xml:"id,attr"`
	ObjectType       string `xml:"ObjectType"`
	Level            string `xml:"Level"`
	Exp              string `xml:"Exp"`
	CurrentFame      string `xml:"CurrentFame"`
	Equipment        string `xml:"Equipment"`
	MaxHitPoints     string `xml:"MaxHitPoints"`
	HitPoints        string `xml:"HitPoints"`
	MaxMagicPoints   string `xml:"MaxMagicPoints"`
	Attack           string `xml:"Attack"`
	Defense          string `xml:"Defense"`
	Speed            string `xml:"Speed"`
	Dexterity        string `xml:"Dexterity"`
	HpRegen          string `xml:"HpRegen"`
	MpRegen          string `xml:"MpRegen"`
	HealthStackCount string `xml:"HealthStackCount"`
	MagicStackCount  string `xml:"MagicStackCount"`
	Dead             string `xml:"Dead"`
	PCStats          string `xml:"PCStats"`
	Account          struct {
		Name string `xml:"Name"`
	} `xml:"Account"`
	HasBackpack string `xml:"HasBackpack"`
}
