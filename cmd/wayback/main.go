package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/whosonfirst/go-whosonfirst-geojson-v2/feature"
	"github.com/whosonfirst/go-whosonfirst-index"
	"github.com/aaronland/go-internetarchive/wayback"
	"io"
	"log"
)

func main() {

	template := flag.String("template", "", "...")
	mode := flag.String("mode", "repo", "...")

	flag.Parse()

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

		// log.Println("SAVE", url)
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
