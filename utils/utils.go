package utils

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var (
	lockfilePath string
	lockfile     *os.File
)

func CreateLock(filename string) (*os.File, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		lockfilePath = filename
		lockfile, err = os.OpenFile(filename, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0666)
		return lockfile, err
	}
	return nil, errors.New("an instance is already running")
}

func RemoveLock() {
	lockfile.Close()
	os.Remove(lockfilePath)
}

func Unzip(src string, dest string) ([]string, error) {

	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()

		if err != nil {
			return filenames, err
		}
	}
	return filenames, nil
}

func Copy(src, dst string) error {
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		in, err := os.Open(src)
		if err != nil {
			return err
		}
		defer in.Close()

		out, err := os.Create(dst)
		if err != nil {
			return err
		}
		defer out.Close()

		_, err = io.Copy(out, in)
		if err != nil {
			return err
		}
	}
	return nil
}
