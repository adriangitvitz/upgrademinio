package utils

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

const (
	releaseTagTimeLayout       = "2006-01-02T15-04-05Z"
	releaseTagTimeLayoutBackup = "2006-01-02T15:04:05Z"
)

func ParseReleaseTag(releaseTag string) (time.Time, error) {
	fields := strings.Split(releaseTag, ".")
	if len(fields) < 1 {
		return time.Time{}, fmt.Errorf("invalid release tag: %s", releaseTag)
	}
	releaseTimeStr := fields[0]
	if len(fields) > 1 {
		releaseTimeStr = fields[1]
	}
	releaseTime, err := time.Parse(releaseTagTimeLayout, releaseTimeStr)
	if err != nil {
		return time.Parse(releaseTagTimeLayoutBackup, releaseTimeStr)
	}
	return releaseTime, nil
}

func FindInSlice(slice []string, val string) (string, bool) {
	for _, item := range slice {
		if item == val {
			return item, true
		}
	}
	return "", false
}

func ExtractFile(tr *tar.Reader, basePath string, name string) error {
	outFile, err := os.Create(path.Join(basePath, path.Base(name)))
	if err != nil {
		return fmt.Errorf("error creating file: %s: %w", name, err)
	}
	defer outFile.Close()
	if _, err := io.Copy(outFile, tr); err != nil {
		return fmt.Errorf("error writing to file: %s: %w", name, err)
	}
	return nil
}

func ExtractTar(filesToExtract []string, basePath, tarFileName string) error {
	tarFile, err := os.Open(path.Join(basePath, tarFileName))
	if err != nil {
		return fmt.Errorf("error opening tar file: %w", err)
	}
	defer tarFile.Close()

	var tr *tar.Reader
	if strings.HasSuffix(tarFileName, ".gz") {
		gz, err := gzip.NewReader(tarFile)
		if err != nil {
			return fmt.Errorf("error in gzip reader: %w", err)
		}
		defer gz.Close()
		tr = tar.NewReader(gz)
	} else {
		tr = tar.NewReader(tarFile)
	}

	remaining := len(filesToExtract)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			if remaining == 0 {
				return nil
			}
			return fmt.Errorf("EOF reached before extracting")
		}
		if err != nil {
			return fmt.Errorf("reading tar: %w", err)
		}

		if header.Typeflag == tar.TypeReg {
			if name, found := FindInSlice(filesToExtract, header.Name); found {
				if err := ExtractFile(tr, basePath, name); err != nil {
					return err
				}
				remaining--
			}
		}
	}
}

func FetchImage(imageName string) (v1.Image, error) {
	ref, err := name.ParseReference(imageName)
	if err != nil {
		return nil, fmt.Errorf("error parsing image name: %w", err)
	}
	img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return nil, fmt.Errorf("error fetching image: %w", err)
	}
	return img, nil
}

func FindLargestLayer(img v1.Image) (string, int64, error) {
	layers, err := img.Layers()
	if err != nil {
		return "", 0, fmt.Errorf("error getting layers: %w", err)
	}

	start := 0
	if len(layers) >= 2 {
		start = 1
	}

	maxSizeHash, _ := layers[start].Digest()
	maxSize, _ := layers[start].Size()

	for i := range layers[start+1:] {
		size, _ := layers[i].Size()
		if size > maxSize {
			maxSize = size
			maxSizeHash, _ = layers[i].Digest()
		}
	}

	return strings.Split(maxSizeHash.String(), ":")[1], maxSize, nil
}
