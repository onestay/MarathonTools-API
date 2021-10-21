package donations

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"

	"github.com/onestay/MarathonTools-API/api/common"
)

// DonationProvider is the interface that has to be satisfied for something to work as an donation provider
type DonationProvider interface {
	// GetTotalAmount should return the total donation amount as a float64. It shouldn't be returned as cents
	GetTotalAmount() (float64, error)
	// GetTotalDonations should return the number of total donations
	GetTotalDonations() (int, error)
	// GetDonations should return all Donations
	GetDonations() ([]Donation, error)
}

// Donation represents a single donation
// Non initialized fields will not be send to the client
type Donation struct {
	Amount  float64   `json:"amount,omitempty"`
	Message string    `json:"message,omitempty"`
	Name    string    `json:"name,omitempty"`
	Created time.Time `json:"created,omitempty"`
	User    string    `json:"user,omitempty"`
}

// DonationController represents the donation controller
type DonationController struct {
	base          *common.Controller
	d             DonationProvider
	t             *time.Ticker
	donationTotal float64
	enabled       bool
}

// NewDonationController takes the base controller and an donation interface and returns a new DonationController
func NewDonationController(b *common.Controller, d DonationProvider, e bool) *DonationController {
	dController := &DonationController{
		base:    b,
		d:       d,
		enabled: e,
	}

	if !e {
		return dController
	}
	t, _ := dController.d.GetTotalAmount()

	dController.donationTotal = t

	return dController
}

// GetTotal will get the total amount of money donated
func (d *DonationController) GetTotal(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if !d.enabled {
		d.base.Response("", "Donations have not been enabled.", http.StatusBadRequest, w)
		return
	}

	amount, err := d.d.GetTotalAmount()
	if err != nil {
		d.base.Response("", "An error occured getting total donation amount", 500, w)
		return
	}

	res := struct {
		DonationAmount float64 `json:"donationAmount"`
	}{amount}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// GetAll will return all donations
func (d *DonationController) GetAll(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if !d.enabled {
		d.base.Response("", "Donations have not been enabled.", http.StatusBadRequest, w)
		return
	}

	donations, err := d.d.GetDonations()
	if err != nil {
		d.base.Response("", "An error occured getting donations", 500, w)
		return
	}

	res := struct {
		Donations []Donation `json:"donations"`
	}{donations}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// GetTotalDonations will return the number of all donations
func (d *DonationController) GetTotalDonations(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if !d.enabled {
		d.base.Response("", "Donations have not been enabled.", http.StatusBadRequest, w)
		return
	}

	amount, err := d.d.GetTotalDonations()
	if err != nil {
		d.base.Response("", "An error occured getting donations", 500, w)
		return
	}

	res := struct {
		TotalDonations int `json:"totalDonations"`
	}{amount}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (d *DonationController) StartTotalUpdate(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if !d.enabled {
		d.base.Response("", "Donations have not been enabled.", http.StatusBadRequest, w)
		return
	}

	if d.t != nil {
		d.base.Response("", "already running", 400, w)
		return
	}
	interval := 5
	a := r.URL.Query().Get("interval")
	if i, err := strconv.Atoi(a); err != nil && i > 0 {
		interval = i
	}

	d.t = time.NewTicker(time.Duration(interval) * time.Second)

	go func() {
		for {
			<-d.t.C
			t, err := d.d.GetTotalAmount()
			if err != nil {
				d.base.LogError("while getting donation total", err, false)
			}
			d.base.WSDonationUpdate(d.donationTotal, t)
			d.donationTotal = t
		}
	}()

	w.WriteHeader(http.StatusNoContent)

}

func (d *DonationController) StopTotalUpdate(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if !d.enabled {
		d.base.Response("", "Donations have not been enabled.", http.StatusBadRequest, w)
		return
	}

	if d.t == nil {
		d.base.Response("", "not running", 400, w)
		return
	}
	d.t.Stop()
	d.t = nil
	w.WriteHeader(http.StatusNoContent)
}
