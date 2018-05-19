package common

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

// Checklist provides the implamentation of a checklist
type Checklist struct {
	Items map[string]bool
	b     *Controller
}

// NewChecklist initizalies and returns a new Checklist
func NewChecklist(b *Controller) *Checklist {
	return &Checklist{
		Items: make(map[string]bool, 100),
		b:     b,
	}
}

// AddItem will add an item to the checklist
func (c *Checklist) AddItem(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if i := r.URL.Query().Get("item"); len(i) != 0 {
		c.Items[i] = false

		json.NewEncoder(w).Encode(c.Items)
		return
	}
	c.b.Response("", "no item defined", http.StatusBadRequest, w)
}

// DeleteItem will delete an item from the checklist
func (c *Checklist) DeleteItem(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if i := r.URL.Query().Get("item"); len(i) != 0 {
		delete(c.Items, i)

		json.NewEncoder(w).Encode(c.Items)
		return
	}
	c.b.Response("", "no item defined", http.StatusBadRequest, w)
}

// ToggleItem will toggle the status of an item if it exists
func (c *Checklist) ToggleItem(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if i := r.URL.Query().Get("item"); len(i) != 0 {
		r, e := c.Items[i]
		if e {
			c.Items[i] = !r
		}

		json.NewEncoder(w).Encode(c.Items)
		return
	}
	c.b.Response("", "no item defined", http.StatusBadRequest, w)
}

// CheckDoneHTTP will return whether all the items in the checklist are done
func (c *Checklist) CheckDoneHTTP(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	c.b.Response(strconv.FormatBool(c.CheckDone()), "", http.StatusOK, w)
}

// CheckDone will return whether all the items in the checklist are done
func (c *Checklist) CheckDone() bool {
	for _, v := range c.Items {
		if !v {
			return false
		}
	}
	return true
}

//GetChecklist will get all items from the checlist
func (c *Checklist) GetChecklist(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	json.NewEncoder(w).Encode(c.Items)
}
