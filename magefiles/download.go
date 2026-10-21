package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
)

func Download(ctx context.Context) error {
	err := download(
		"https://registry.npmjs.org/@pierre/highlights/-/highlights-0.2.0.tgz",
		"./bin/highlights.tgz",
	)
	if err != nil {
		return err
	}

	err = extract(
		"./bin/highlights.tgz",
		spec{
			destination: "./internal/frontend/static/highlights",
			processor: func(path string) *string {
				switch path {
				case "package/dist/themes/ayu-light.js", "package/dist/themes/ayu-dark.js":
					return ptr(strings.TrimPrefix(path, "package/dist/"))
				case "package/dist/themes.js", "package/dist/workerd.js":
					return nil
				}
				// only the es modules needed by the browser; skip themes, types and sourcemaps
				if strings.HasPrefix(path, "package/dist/") &&
					!strings.HasPrefix(path, "package/dist/themes/") &&
					(strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".mjs")) {
					return ptr(strings.TrimPrefix(path, "package/dist/"))
				}
				return nil
			},
		},
	)
	if err != nil {
		return err
	}

	err = download(
		"https://unpkg.com/htmx.org@2.0.4",
		"./internal/frontend/static/htmx.js",
	)
	if err != nil {
		return err
	}

	return nil
}

type spec struct {
	destination string
	processor   func(path string) *string
}

func download(url, destination string) error {
	logstep(fmt.Sprintf("downloading %s to %s", url, destination))
	var err error

	start := time.Now()
	defer func() {
		elapsed := time.Since(start).Round(time.Millisecond)
		if err != nil {
			color.Red(" ✘ %s", elapsed)
			return
		}
		color.Green(" ✔ %s", elapsed)
	}()

	if _, err := os.Stat(destination); err == nil {
		return nil
	}

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func extract(filename string, spec spec) error {
	logstep(fmt.Sprintf("extracting %s", filename))
	var err error

	start := time.Now()
	defer func() {
		elapsed := time.Since(start).Round(time.Millisecond)
		if err != nil {
			color.Red(" ✘ %s", elapsed)
			return
		}
		color.Green(" ✔ %s", elapsed)
	}()

	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	decompressor, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer decompressor.Close()

	reader := tar.NewReader(decompressor)

	if err := os.MkdirAll(spec.destination, 0o755); err != nil {
		return err
	}

	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		processed := spec.processor(header.Name)
		if processed == nil {
			continue
		}
		target := filepath.Join(spec.destination, *processed)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.Create(target)
			if err != nil {
				return err
			}
			defer out.Close()

			if _, err := io.Copy(out, reader); err != nil {
				return err
			}
		}
	}

	return nil
}

func logstep(text string) {
	fmt.Println(
		"\n",
		color.MagentaString(">"),
		color.New(color.Bold).Sprint(text),
	)
}
