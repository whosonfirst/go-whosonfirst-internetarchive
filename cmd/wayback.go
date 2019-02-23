package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/whosonfirst/go-whosonfirst-geojson-v2/feature"
	"github.com/whosonfirst/go-whosonfirst-index"
	"io"
	"io/ioutil"
	"log"
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

func main() {

	template := flag.String("template", "", "...")
	mode := flag.String("mode", "repo", "...")

	flag.Parse()

	rate := time.Second / 10
	throttle := time.Tick(rate)

	needs_archive := func(ctx context.Context, wof_url string) (bool, error) {

		<-throttle

		select {
		case <-ctx.Done():
			return false, nil
		default:
			// pass
		}

		url := fmt.Sprintf("http://archive.org/wayback/available?url=%s", wof_url)

		rsp, err := http.Get(url)

		if err != nil {
			return false, err
		}

		defer rsp.Body.Close()

		if rsp.StatusCode != 200 {
			log.Println(url, rsp.Status)
			return false, nil
		}
		
		body, err := ioutil.ReadAll(rsp.Body)

		if err != nil {
			return false, err
		}

		var arch Archive

		err = json.Unmarshal(body, &arch)

		if err != nil {
			return false, err
		}

		c := arch.Snapshots.Closest

		if c != nil {
			return false, nil
		}

		return true, nil
	}

	archive := func(ctx context.Context, wof_url string) error {

		<-throttle

		select {
		case <-ctx.Done():
			return nil
		default:
			// pass
		}

		url := fmt.Sprintf("https://web.archive.org/save/%s", wof_url)

		rsp, err := http.Get(url)

		if err != nil {
			return err
		}

		defer rsp.Body.Close()

		if rsp.StatusCode != 200 {
			return errors.New(rsp.Status)
		}
		
		return err
	}

	cb := func(fh io.Reader, ctx context.Context, args ...interface{}) error {

		path, err := index.PathForContext(ctx)

		if err != nil {
			return err
		}

		f, err := feature.LoadGeoJSONFeatureFromReader(fh)

		if err != nil {
			msg := fmt.Sprintf("Unable to load %s (%s)", path, err)
			return errors.New(msg)
		}

		id := f.Id()
		wof_url := fmt.Sprintf(*template, id)

		do_archive, err := needs_archive(ctx, wof_url)

		log.Println(id, do_archive)

		if err != nil {
			return err
		}

		if !do_archive {
			return nil
		}

		return archive(ctx, wof_url)
	}

	i, err := index.NewIndexer(*mode, cb)

	if err != nil {
		log.Fatal(err)
	}

	paths := flag.Args()

	err = i.IndexPaths(paths)

	if err != nil {
		log.Fatal(err)
	}

}
