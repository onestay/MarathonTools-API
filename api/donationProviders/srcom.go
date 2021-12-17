package donationProviders

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/onestay/MarathonTools-API/api/routes/donations"
)

type srComMarathon struct {
	Data struct {
		ID    string `json:"id"`
		Names struct {
			International string `json:"international"`
		} `json:"names"`
		Abbreviation string `json:"abbreviation"`
		Weblink      string `json:"weblink"`
		Links        []struct {
			Rel string `json:"rel"`
			URI string `json:"uri"`
		} `json:"links"`
	} `json:"data"`
}

type srComDonation struct {
	Data []struct {
		ID      string    `json:"id"`
		Created time.Time `json:"created"`
		Amount  int       `json:"amount"`
		Comment string    `json:"comment"`
		Status  string    `json:"status"`
		Links   []struct {
			Rel string `json:"rel"`
			URI string `json:"uri"`
		} `json:"links"`
		User struct {
			Data struct {
				Names struct {
					International string `json:"international"`
				} `json:"names"`
			} `json:"data"`
		} `json:"user"`
	} `json:"data"`
	Pagination struct {
		Offset int `json:"offset"`
		Max    int `json:"max"`
		Size   int `json:"size"`
		Links  []struct {
			Rel string `json:"rel"`
			URI string `json:"uri"`
		} `json:"links"`
	} `json:"pagination"`
}

type donationSummary struct {
	Data struct {
		TotalDonations int `json:"total-donations"`
		TotalDonated   int `json:"total-donated"`
	} `json:"data"`
}

const (
	baseURL = "https://www.speedrun.com/api/v1/games/"
)

// SRComDonationProvider satisfies the DonationsInterface and can be used as a donation provider
type SRComDonationProvider struct {
	marathonSlug  string
	marathonID    string
	links         map[string]string
	usernameCache map[int]string
	donations     []donations.Donation
}

// NewSRComDonationProvider will initialize and return a new SRCom Donation provider where ms is the marathon slug
func NewSRComDonationProvider(ms string) (*SRComDonationProvider, error) {
	res, err := http.Get(baseURL + ms)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == 404 {
		return nil, errors.New("couldn't find marathon")
	}

	if res.StatusCode != 200 {
		return nil, errors.New("non 200 status code")
	}

	var m srComMarathon
	json.NewDecoder(res.Body).Decode(&m)

	linkMap := make(map[string]string)
	// check if donations are enabled
	donationsEnabled := false
	for _, k := range m.Data.Links {
		if k.Rel == "donation-summary" {
			donationsEnabled = true
			linkMap["summary"] = k.URI
		} else if k.Rel == "donation-list" {
			linkMap["list"] = k.URI
		} else if k.Rel == "donation-goals" {
			linkMap["goals"] = k.URI
		} else if k.Rel == "donation-prizes" {
			linkMap["prizes"] = k.URI
		} else if k.Rel == "donation-bidwars" {
			linkMap["bidwars"] = k.URI
		}
	}

	if !donationsEnabled {
		return nil, errors.New("donations not enabled for this marathon")
	}
	// I think we can assure, that all links are filled out after donationsEnabled has passed
	return &SRComDonationProvider{
		marathonSlug: ms,
		marathonID:   m.Data.ID,
		links:        linkMap,
	}, nil
}

// GetTotalAmount will get the total donation amount
func (sr *SRComDonationProvider) GetTotalAmount() (float64, error) {
	res, err := http.Get(sr.links["summary"])
	if err != nil {
		return -1, err
	}

	if res.StatusCode != 200 {
		return -1, err
	}

	var ds donationSummary

	json.NewDecoder(res.Body).Decode(&ds)

	return float64(ds.Data.TotalDonated) / 100, nil
}

// GetTotalDonations will return the amount of donations
func (sr *SRComDonationProvider) GetTotalDonations() (int, error) {
	res, err := http.Get(sr.links["summary"])
	if err != nil {
		return -1, err
	}

	if res.StatusCode != 200 {
		return -1, err
	}

	var ds donationSummary

	json.NewDecoder(res.Body).Decode(&ds)

	return ds.Data.TotalDonations, nil
}

// GetDonations will return all donations
func (sr *SRComDonationProvider) GetDonations() ([]donations.Donation, error) {
	limit, offset := "200", 200
	var don srComDonation
	var max, size int

	res, err := http.Get(sr.links["list"] + "?max=" + limit + "&embed=user")
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, err
	}

	json.NewDecoder(res.Body).Decode(&don)
	size, max = don.Pagination.Size, don.Pagination.Max
	fmt.Println(size, max)

	for size >= max {
		res, err := http.Get(sr.links["list"] + "?max=" + limit + "&offset=" + strconv.Itoa(offset) + "&embed=user")
		if err != nil {
			return nil, err
		}

		if res.StatusCode != 200 {
			return nil, err
		}

		var pDon srComDonation
		json.NewDecoder(res.Body).Decode(&pDon)

		don.Data = append(don.Data, pDon.Data...)
		size, max = pDon.Pagination.Size, pDon.Pagination.Max
		offset += 200
	}
	ds := make([]donations.Donation, len(don.Data))
	for i, d := range don.Data {
		ds[i].Amount = float64(d.Amount) / 100
		ds[i].Created = d.Created
		ds[i].Message = d.Comment
		ds[i].Name = d.ID
		ds[i].User = d.User.Data.Names.International
	}

	return ds, nil
}
