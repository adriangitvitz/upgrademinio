package handlers

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"upgrademinio/utils"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

type ImgRefs struct {
	Tag       string
	Reference name.Reference
	Img       v1.Image
}

type Binaries struct {
	Minio       string `json:"minio"`
	MinioSha256 string `json:"MinioSha256"`
	Minisig     string `json:"minisig"`
}

type RetrieveContentService interface {
	RetrieveContent(imageName string) (*Binaries, error)
	GetBinaries(name string, tag string) (string, error)
	Close()
}

type RetrieveContentServiceImpl struct {
	basePath string
	cache    *utils.LRUCache[string]
}

func NewRetrieveService(basePath string) RetrieveContentService {
	return &RetrieveContentServiceImpl{
		basePath: basePath,
		cache:    utils.NewLRUCache[string](20, 10*time.Minute),
	}
}

func GetTag(imageName string) (*ImgRefs, error) {
	ref, err := name.ParseReference(imageName)
	if err != nil {
		return nil, err
	}

	img, err := utils.FetchImage(imageName)
	if err != nil {
		return nil, err
	}

	cfg, err := img.ConfigFile()
	if err != nil {
		return nil, err
	}

	tag, ok := cfg.Config.Labels["release"]
	if !ok {
		tag, ok = cfg.Config.Labels["version"]
	}

	tag = strings.TrimSpace(tag)
	if !ok || tag == "" {
		return nil, fmt.Errorf("release tag not found")
	}
	return &ImgRefs{
		Tag:       tag,
		Reference: ref,
		Img:       img,
	}, nil
}

func createContentImage(basePath string, refImg *ImgRefs) (*Binaries, string, error) {
	largestLayerHash, _, err := utils.FindLargestLayer(refImg.Img)
	if err != nil {
		return nil, "", err
	}

	hashbasePath := fmt.Sprintf("%s/%s/", basePath, largestLayerHash)
	err = os.MkdirAll(filepath.Dir(hashbasePath), 0777)
	if err != nil {
		return nil, "", err
	}

	imageTarPath := path.Join(hashbasePath, "image.tar")
	f, err := os.Create(imageTarPath)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()

	if err = tarball.Write(refImg.Reference, refImg.Img, f); err != nil {
		return nil, "", err
	}

	fileNameToExtract := largestLayerHash + ".tar.gz"
	if err = utils.ExtractTar([]string{fileNameToExtract}, hashbasePath, "image.tar"); err != nil {
		return nil, "", err
	}

	latestAssets := []string{"opt/bin/minio", "opt/bin/minio.sha256sum", "opt/bin/minio.minisig"}
	legacyAssets := []string{"usr/bin/minio", "usr/bin/minio.sha256sum", "usr/bin/minio.minisig"}

	if err = utils.ExtractTar(latestAssets, hashbasePath, fileNameToExtract); err != nil {
		if err = utils.ExtractTar(legacyAssets, hashbasePath, fileNameToExtract); err != nil {
			return nil, "", err
		}
	}

	if _, err := utils.ParseReleaseTag(refImg.Tag); err != nil {
		return nil, "", err
	}

	filesToRename := map[string]string{
		"minio":           "minio." + refImg.Tag,
		"minio.sha256sum": "minio." + refImg.Tag + ".sha256sum",
		"minio.minisig":   "minio." + refImg.Tag + ".minisig",
	}

	for src, dest := range filesToRename {
		srcPath := path.Join(hashbasePath, src)
		destPath := path.Join(hashbasePath, dest)
		if err := os.Rename(srcPath, destPath); err != nil {
			return nil, "", err
		}
	}

	err = os.Remove(fmt.Sprintf("%s/%s", hashbasePath, fileNameToExtract))
	if err != nil {
		return nil, "", err
	}
	err = os.Remove(fmt.Sprintf("%s/%s", hashbasePath, "image.tar"))
	if err != nil {
		return nil, "", err
	}

	return &Binaries{
		Minio:       fmt.Sprintf("%s", "minio."+refImg.Tag),
		MinioSha256: fmt.Sprintf("%s", "minio."+refImg.Tag+".sha256sum"),
		Minisig:     fmt.Sprintf("%s", "minio."+refImg.Tag+".minisig"),
	}, largestLayerHash, nil
}

func (r *RetrieveContentServiceImpl) RetrieveContent(imageName string) (*Binaries, error) {
	refImg, err := GetTag(imageName)
	if err != nil {
		return nil, err
	}
	value, exists := r.cache.Get(refImg.Tag)
	if exists {
		basePath := fmt.Sprintf("%s/%s/", r.basePath, value)
		_, err := os.Stat(basePath)
		if err != nil {
			return nil, err
		}
		return &Binaries{
			Minio:       fmt.Sprintf("%s", "minio."+refImg.Tag),
			MinioSha256: fmt.Sprintf("%s", "minio."+refImg.Tag+".sha256sum"),
			Minisig:     fmt.Sprintf("%s", "minio."+refImg.Tag+".minisig"),
		}, nil
	}
	data, hash, err := createContentImage(r.basePath, refImg)
	if err == nil {
		r.cache.Set(refImg.Tag, hash)
	}
	return data, err
}

func (r *RetrieveContentServiceImpl) GetBinaries(name string, tag string) (string, error) {
	value, exists := r.cache.Get(tag)
	binaryPath := ""
	if exists {
		binaryPath = fmt.Sprintf("%s/%s/%s", r.basePath, value, name)
		return binaryPath, nil
	}
	return "", fmt.Errorf("%s not found", name)
}

func (r *RetrieveContentServiceImpl) Close() {
	r.cache.Close()
}
