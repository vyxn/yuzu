package zip

import (
	"archive/zip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

func WriteInZip(
	zipPath string,
	pathInZip string,
	r io.Reader,
) (ferr error) {
	// Read original zip
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer func() {
		if err := zr.Close(); err != nil {
			ferr = errors.Join(ferr, err)
			return
		}
	}()

	// Create a tmp zip file
	dir := filepath.Dir(zipPath)
	tmpFile, err := os.CreateTemp(dir, "*.zip.tmp")
	if err != nil {
		return err
	}
	defer func() {
		if err := tmpFile.Close(); err != nil {
			ferr = errors.Join(ferr, err)
			return
		}
	}()

	zw := zip.NewWriter(tmpFile)
	defer func() {
		if err := zw.Close(); err != nil {
			ferr = errors.Join(ferr, err)
			return
		}
	}()

	// Copy old files
	for _, f := range zr.File {
		if f.Name == pathInZip {
			// Skip old version
			continue
		}

		// Copy entry as-is
		err = copyZipFile(f, zw)
		if err != nil {
			return err
		}
	}

	// 4. Add the new file under the same path.
	wf, err := zw.Create(pathInZip)
	if err != nil {
		return err
	}
	if _, err := io.Copy(wf, r); err != nil {
		return err
	}

	// 6. Replace the original zip atomically.
	return os.Rename(tmpFile.Name(), zipPath)
}

func copyZipFile(
	f *zip.File,
	zw *zip.Writer,
) (ferr error) {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer func() {
		if err := rc.Close(); err != nil {
			ferr = errors.Join(ferr, err)
			return
		}
	}()

	hdr := f.FileHeader
	hdr.Method = f.Method
	w, err := zw.CreateHeader(&hdr)
	if err != nil {
		return err
	}
	if _, err := io.Copy(w, rc); err != nil {
		return err
	}

	return nil
}

func MatchInZip(zipPath string, re *regexp.Regexp) (bool, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return false, err
	}
	defer r.Close()

	for _, f := range r.File {
		if re.MatchString(f.Name) {
			return true, nil
		}
	}
	return false, nil
}
