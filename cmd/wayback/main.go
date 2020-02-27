package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/whosonfirst/go-whosonfirst-geojson-v2/feature"
	"github.com/whosonfirst/go-whosonfirst-index"
	_ "github.com/whosonfirst/go-whosonfirst-index/fs"	
	"github.com/aaronland/go-internetarchive/wayback"
	"github.com/sfomuseum/go-url-unshortener"	
	"io"
	"log"
	"time"
)

func main() {

	template := flag.String("template", "", "...")
	mode := flag.String("mode", "repo://", "...")
	unshorten := flag.Bool("unshorten", false, "...")
	qps := flag.Int("qps", 10, "Number of (unshortening) queries per second")
	to := flag.Int("timeout", 30, "Maximum number of seconds of for an unshorterning request")

	dryrun := flag.Bool("dryrun", false, "...")
	flag.Parse()

	var worker unshortener.Unshortener

	if *unshorten {

		rate := time.Second / time.Duration(*qps)
		timeout := time.Second * time.Duration(*to)
		
		w, err := unshortener.NewThrottledUnshortener(rate, timeout)

		if err != nil {
			log.Fatalf("Failed to create unshortener, %v", err)
		}

		worker = w
	}
	
	opts, err := wayback.DefaultWaybackMachineOptions()

	if err != nil {
		log.Fatal(err)
	}

	wb, err := wayback.NewWaybackMachine(opts)

	if err != nil {
		log.Fatal(err)
	}

	cb := func(ctx context.Context, fh io.Reader, args ...interface{}) error {

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
		url := fmt.Sprintf(*template, id)

		u, err := unshortener.UnshortenString(ctx, worker, url)

		if err != nil {
			log.Printf("Failed to unshorten '%s', %v\n", url, err)
			return nil
		}

		url = u.String()

		if *dryrun {
			log.Printf("[dryrun] save %s\n", url)
			return nil
		}
				
		// TO DO: get wof:lastmodified and use it to call wb.HasArchiveNewerThan(ctx, url, t)
		// (20190225/thisisaaronland)
		
		do_archive, err := wb.HasArchive(ctx, url)

		if err != nil {
			return err
		}

		if !do_archive {
			// log.Println("SKIP", url)
			return nil
		}

		log.Printf("Saved '%s'\n", url)
		return wb.Save(ctx, url)
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
