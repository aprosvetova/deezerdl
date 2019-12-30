package internal

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/docopt/docopt-go"
	"github.com/joshbarrass/deezerdl/pkg/deezer"
	"github.com/joshbarrass/deezerdl/pkg/writetracker"
	"github.com/sirupsen/logrus"
)

// https://progolang.com/how-to-download-files-in-go/

func DownloadFile(url, outPath string) error {
	outFile, err := os.Create(outPath + ".part")
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Get the file
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(outFile, io.TeeReader(resp.Body, &writetracker.WriteTracker{}))
	if err != nil {
		return err
	}

	// move to new line because of how ShowProgress works
	fmt.Println("")

	// rename part file
	err = os.Rename(outPath+".part", outPath)
	if err != nil {
		return err
	}

	return nil
}

// Download reads arguments from docopt options to work out what to
// download
func Download(opts *docopt.Opts, config *Configuration) {
	// get the ID
	ID, err := opts.Int("<ID>")
	if err != nil {
		logrus.Fatalf("failed to parse arguments: %s", err)
	}

	// make API
	api, err := deezer.NewAPI(false)
	if err != nil {
		logrus.Fatalf("failed to create api: %s", err)
	}

	// log in
	if err := api.CookieLogin(config.ARLCookie); err != nil {
		logrus.Fatalf("failed to log in: %s", err)
	}

	// check track
	if track, err := opts.Bool("track"); err != nil {
		logrus.Fatalf("failed to parse arguments: %s", err)
	} else if track {
		if err := downloadTrack(ID, api); err != nil {
			logrus.Fatalf("failed to download track: %s", err)
		}
	}

}

// downloadTrack is for downloading an individual track
func downloadTrack(ID int, api *deezer.API) error {
	// get track info
	track, err := api.GetSongData(ID)
	if err != nil {
		return err
	}

	// get the download URL
	downloadUrl, err := track.GetDownloadURL(deezer.FLAC)
	if err != nil {
		return err
	}

	filename := track.Title + ".flac"
	encFilename := filename + ".enc"

	// download file
	if err := DownloadFile(downloadUrl.String(), encFilename); err != nil {
		return err
	}
	defer os.Remove(encFilename)

	// decrypt song
	key := track.GetBlowfishKey()
	err = deezer.DecryptSongFile(key, encFilename, filename)
	if err != nil {
		return err
	}

	return nil
}