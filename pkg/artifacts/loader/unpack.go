package loader

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/util/errors"
)

type artifact struct {
	rawData []byte
	commit  string
}

// LoadZippedArtifactsFromDirectory takes the directory of the artifacts and unwraps the zip files in it.
// Artifact zip files are expected to be in a flat directory.
// Failed to unwrap an artifact does not stop the entire operation.
// The results will just not be included in the artifact.
func LoadZippedArtifactsFromDirectory(dir string) ([]artifact, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var errs []error
	var artifacts []artifact

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		ar, err := unwrapArtifactZip(filepath.Join(dir, f.Name()))
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to unwrap %s, %v", f.Name(), err))
			continue
		}
		artifacts = append(artifacts, *ar)

	}
	return artifacts, errors.NewAggregate(errs)
}

func unwrapArtifactZip(file string) (*artifact, error) {
	name := filepath.Base(file)
	splits := strings.Split(name, "-")
	if len(splits) < 2 {
		return nil, fmt.Errorf("artifact is not following the formate <test-name>-<commit>-<run id>")
	}

	raw, err := unzip(file)
	if err != nil {
		return nil, err
	}

	return &artifact{
		commit:  splits[len(splits)-2],
		rawData: raw,
	}, nil
}

func unzip(src string) ([]byte, error) {
	r, err := zip.OpenReader(src)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var raw [][]byte

	for _, f := range r.File {

		if f.FileInfo().IsDir() {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return nil, err
		}

		file, err := ioutil.ReadAll(rc)
		if err != nil {
			return nil, err
		}

		rc.Close()

		raw = append(raw, file)
	}
	return bytes.Join(raw, []byte{}), nil
}
