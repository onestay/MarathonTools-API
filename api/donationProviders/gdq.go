package donationProviders

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/onestay/MarathonTools-API/api/routes/donations"

	"github.com/PuerkitoBio/goquery"

	"golang.org/x/net/publicsuffix"
)

type gdqDonation []struct {
	Pk     int    `json:"pk"`
	Model  string `json:"model"`
	Fields struct {
		Comment               string    `json:"comment"`
		Domain                string    `json:"domain"`
		Readstate             string    `json:"readstate"`
		DonorAddresszip       string    `json:"donor__addresszip"`
		Requestedemail        string    `json:"requestedemail"`
		DonorFirstname        string    `json:"donor__firstname"`
		DonorAlias            string    `json:"donor__alias"`
		Commentlanguage       string    `json:"commentlanguage"`
		DonorVisibility       string    `json:"donor__visibility"`
		Bidstate              string    `json:"bidstate"`
		Event                 int       `json:"event"`
		DonorAddressstreet    string    `json:"donor__addressstreet"`
		DonorLastname         string    `json:"donor__lastname"`
		Fee                   string    `json:"fee"`
		Testdonation          bool      `json:"testdonation"`
		DonorEmail            string    `json:"donor__email"`
		Timereceived          time.Time `json:"timereceived"`
		Donor                 int       `json:"donor"`
		Public                string    `json:"public"`
		DonorSolicitemail     string    `json:"donor__solicitemail"`
		Transactionstate      string    `json:"transactionstate"`
		Requestedalias        string    `json:"requestedalias"`
		DonorAddressstate     string    `json:"donor__addressstate"`
		DomainID              string    `json:"domainId"`
		Currency              string    `json:"currency"`
		Commentstate          string    `json:"commentstate"`
		Modcomment            string    `json:"modcomment"`
		Amount                string    `json:"amount"`
		DonorAddresscity      string    `json:"donor__addresscity"`
		Requestedsolicitemail string    `json:"requestedsolicitemail"`
		DonorPaypalemail      string    `json:"donor__paypalemail"`
		DonorPublic           string    `json:"donor__public"`
		Requestedvisibility   string    `json:"requestedvisibility"`
	} `json:"fields,omitempty"`
}

type gdqSearch struct {
	Count struct {
		Donors int `json:"donors"`
		Runs   int `json:"runs"`
		Bids   int `json:"bids"`
		Prizes int `json:"prizes"`
	} `json:"count"`
	Agg struct {
		Count  int     `json:"count"`
		Max    string  `json:"max"`
		Amount string  `json:"amount"`
		Avg    float64 `json:"avg"`
		Target string  `json:"target"`
	} `json:"agg"`
}

// GDQDonationProvider represents a GDQDonationProvider
type GDQDonationProvider struct {
	trackerURL, eventID, username, password string
	statsURL, apiURL, loginURL              string
	client                                  http.Client
}

// NewGDQDonationProvider will initialize and return a new GDQ Tracker Donation provider where t is the tracker URL and e is the event id
func NewGDQDonationProvider(t, e, username, password string) (*GDQDonationProvider, error) {
	res, err := http.Get(t + e + "?json")
	if err != nil {
		return nil, errors.New("Couldn't find tracker")
	}

	if res.StatusCode != 200 {
		return nil, errors.New("Couldn't find specified event")
	}

	res, err = http.Get(t + "admin/login")
	if err != nil {
		return nil, errors.New("Couldn't find login page")
	}

	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, err
	}

	client := http.Client{
		Jar: jar,
	}

	gdq := GDQDonationProvider{
		trackerURL: t,
		eventID:    e,
		username:   username,
		password:   password,
		loginURL:   t + "admin/login/",
		statsURL:   t + e + "?json",
		apiURL:     t + "/search?event=" + e,
		client:     client,
	}
	gdq.login()
	gdq.GetDonations()
	// tokenizer := html.NewTokenizer(res.Body)

	return &gdq, nil
}

func (gdq *GDQDonationProvider) login() error {
	res, err := gdq.client.Get(gdq.loginURL)
	if err != nil {
		return errors.New("Couldn't find login page")
	}

	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return err
	}

	var csrftoken string
	for _, attr := range doc.Find("#login-form > input[name=\"csrfmiddlewaretoken\"]").Nodes[0].Attr {
		if attr.Key == "value" {
			csrftoken = attr.Val
		}
	}

	form := url.Values{}
	form.Add("username", gdq.username)
	form.Add("password", gdq.password)
	form.Add("csrfmiddlewaretoken", csrftoken)
	req, err := http.NewRequest("POST", gdq.loginURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}

	req.Header.Add("Referer", gdq.loginURL)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resLogin, err := gdq.client.Do(req)
	if err != nil {
		return err
	}
	if resLogin.StatusCode != 200 {
		return errors.New("non 200 status code")
	}

	return nil

}

// GetTotalAmount will get the total donation amount
func (gdq *GDQDonationProvider) GetTotalAmount() (float64, error) {
	search, err := gdq.getEventInfo()
	if err != nil {
		return -1, err
	}

	amount, err := strconv.ParseFloat(search.Agg.Amount, 64)
	if err != nil {
		return -1, err
	}

	return amount, nil
}

// GetTotalDonations will return the amount of donations
func (gdq *GDQDonationProvider) GetTotalDonations() (int, error) {
	search, err := gdq.getEventInfo()
	if err != nil {
		return -1, err
	}

	return search.Agg.Count, nil
}
func (gdq *GDQDonationProvider) getEventInfo() (*gdqSearch, error) {
	res, err := http.Get(gdq.trackerURL + gdq.eventID + "?json")
	if err != nil || res.StatusCode != 200 {
		return nil, err
	}

	var search gdqSearch

	defer res.Body.Close()

	json.NewDecoder(res.Body).Decode(&search)

	return &search, nil
}

// GetDonations will return all donations
func (gdq *GDQDonationProvider) GetDonations() ([]donations.Donation, error) {
	res, err := gdq.client.Get(gdq.apiURL + "&type=donation")
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, errors.New("Non 200 status code")
	}

	defer res.Body.Close()

	var donData gdqDonation

	json.NewDecoder(res.Body).Decode(&donData)

	ds := make([]donations.Donation, len(donData))

	for i, d := range donData {
		a, err := strconv.ParseFloat(d.Fields.Amount, 64)
		if err != nil {
			return nil, err
		}

		ds[i].Amount = a
		ds[i].Created = d.Fields.Timereceived
		ds[i].Message = d.Fields.Comment
		ds[i].Name = d.Fields.DonorAlias
		ds[i].User = d.Fields.DonorAlias
	}

	return ds, nil
}
