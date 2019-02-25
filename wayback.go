package internetarchive

// this should go in a standalone package somewhere...
// (20190225/thisisaaronland)

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	_ "log"
	"net/http"
	"time"
)

// https://help.archive.org/hc/en-us/articles/360001513491-Save-Pages-in-the-Wayback-Machine
// https://archive.org/help/wayback_api.php

type Archive struct {
	URL       string     `json:"url"`
	Snapshots *Snapshots `json:"archived_snapshots"`
}

type Snapshots struct {
	Closest *Snapshot `json:"closest,omitempty"`
}

type Snapshot struct {
	Available bool   `json:"available"`
	URL       string `json:"url"`
	Timestamp string `json:"timestamp"`
	Status    string `json:"status"`
}

type WaybackMachineOptions struct {
	Throttle <-chan time.Time
	Retries  int
}

func DefaultWaybackMachineOptions() (*WaybackMachineOptions, error) {

	rate := time.Second / 10
	throttle := time.Tick(rate)

	opts := WaybackMachineOptions{
		Throttle: throttle,
		Retries:  5,
	}

	return &opts, nil
}

type WaybackMachine struct {
	Options *WaybackMachineOptions
}

func NewWaybackMachine(opts *WaybackMachineOptions) (*WaybackMachine, error) {

	m := WaybackMachine{
		Options: opts,
	}

	return &m, nil
}

func (m *WaybackMachine) Save(ctx context.Context, url string) error {

	select {
	case <-ctx.Done():
		return nil
	default:
		// pass
	}

	archive_url := fmt.Sprintf("https://web.archive.org/save/%s", url)

	rsp, err := m.get(archive_url)

	if err != nil {
		return err
	}

	defer rsp.Body.Close()

	return err
}

func (m *WaybackMachine) Archives(ctx context.Context, url string) (*Archive, error) {

	select {
	case <-ctx.Done():
		return nil, nil
	default:
		// pass
	}

	archive_url := fmt.Sprintf("http://archive.org/wayback/available?url=%s", url)

	rsp, err := m.get(archive_url)

	if err != nil {
		return nil, err
	}

	defer rsp.Body.Close()

	body, err := ioutil.ReadAll(rsp.Body)

	if err != nil {
		return nil, err
	}

	var arch Archive

	err = json.Unmarshal(body, &arch)

	if err != nil {
		return nil, err
	}

	return &arch, nil
}

func (m *WaybackMachine) HasArchive(ctx context.Context, url string) (bool, error) {

	arch, err := m.Archives(ctx, url)

	if err != nil {
		return false, err
	}

	c := arch.Snapshots.Closest

	if c != nil {
		return false, nil
	}

	return true, nil
}

func (m *WaybackMachine) get(url string) (*http.Response, error) {

	attempts := m.Options.Retries
	delay := 1

	var rsp *http.Response
	var err error

	for attempts > 0 {

		<-m.Options.Throttle

		attempts -= 1

		rsp, err = http.Get(url)

		if err == nil && rsp.StatusCode == 200 {
			break
		}

		time.Sleep(time.Second * time.Duration(delay))
		delay += delay
	}

	return rsp, err
}