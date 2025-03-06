package tftp

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

func (root *Root) unzip(specUrl string) error {

	if root.Exists("RPI_EFI.fd") {
		return nil
	}

	resp, err := http.Get(specUrl)
	if err != nil {
		fmt.Printf("err: %s", err)
	}

	defer resp.Body.Close()
	fmt.Println("status", resp.Status)
	if resp.StatusCode != 200 {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	r, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		if f.FileInfo().IsDir() {
			root.MkdirAll(f.Name, f.Mode())
		} else {
			var fdir string
			if lastIndex := strings.LastIndex(f.Name, string(os.PathSeparator)); lastIndex > -1 {
				fdir = f.Name[:lastIndex]
			}

			err = root.MkdirAll(fdir, f.Mode())
			if err != nil {
				log.Fatal(err)
				return err
			}
			f, err := root.OpenFile(
				f.Name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer f.Close()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
