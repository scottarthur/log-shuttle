package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

func StartOutlets(config ShuttleConfig, drops, lost *Counter, stats chan<- NamedValue, inbox <-chan *Batch, batchReturn chan<- *Batch) *sync.WaitGroup {
	outletWaiter := new(sync.WaitGroup)

	for i := 0; i < config.NumOutlets; i++ {
		outletWaiter.Add(1)
		go func() {
			defer outletWaiter.Done()
			outlet := NewOutlet(config, drops, lost, stats, inbox, batchReturn)
			outlet.Outlet()
		}()
	}

	return outletWaiter
}

type HttpOutlet struct {
	inbox       <-chan *Batch
	batchReturn chan<- *Batch
	stats       chan<- NamedValue
	drops       *Counter
	lost        *Counter
	client      *http.Client
	config      ShuttleConfig
}

func NewOutlet(config ShuttleConfig, drops, lost *Counter, stats chan<- NamedValue, inbox <-chan *Batch, batchReturn chan<- *Batch) *HttpOutlet {
	return &HttpOutlet{
		drops:       drops,
		lost:        lost,
		stats:       stats,
		inbox:       inbox,
		batchReturn: batchReturn,
		config:      config,
		client: &http.Client{
			Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: config.SkipVerify},
				ResponseHeaderTimeout: config.Timeout,
				Dial: func(network, address string) (net.Conn, error) {
					return net.DialTimeout(network, address, config.Timeout)
				},
			},
		},
	}
}

// Outlet receives batches from the inbox and submits them to logplex via HTTP.
func (h *HttpOutlet) Outlet() {

	for batch := range h.inbox {

		err := h.post(batch)
		if err != nil {
			fmt.Fprintf(os.Stderr, "post-error=%s\n", err)
			h.lost.Add(uint64(batch.MsgCount))
		}

		h.batchReturn <- batch
	}
}

func (h *HttpOutlet) post(b *Batch) error {
	req, err := http.NewRequest("POST", h.config.OutletURL(), b)
	if err != nil {
		return err
	}

	drops := h.drops.ReadAndReset()
	lost := h.lost.ReadAndReset()
	if lostAndDropped := drops + lost; lostAndDropped > 0 {
		b.WriteDrops(int(lostAndDropped))
	}

	req.ContentLength = int64(b.Len())
	req.Header.Add("Content-Type", "application/logplex-1")
	req.Header.Add("Logplex-Msg-Count", strconv.Itoa(b.MsgCount))
	req.Header.Add("Logshuttle-Drops", strconv.Itoa(int(drops)))
	req.Header.Add("Logshuttle-Lost", strconv.Itoa(int(lost)))
	resp, err := h.timeRequest(req)
	if err != nil {
		return err
	}

	if h.config.Verbose {
		fmt.Printf("at=post status=%d\n", resp.StatusCode)
	}

	resp.Body.Close()
	return nil
}

func (h *HttpOutlet) timeRequest(req *http.Request) (resp *http.Response, err error) {
	defer func(t time.Time) {
		name := "outlet.post"
		if err != nil {
			name += ".failure"
		} else {
			name += ".success"
		}
		h.stats <- NewNamedValue(name, time.Since(t).Seconds())
	}(time.Now())
	return h.client.Do(req)
}
