package archive

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/nicerobot/ssh-tgzx/internal/constants"
)

// Create writes a tar.gz archive of the given paths to w.
func Create(w io.Writer, paths []string) error {
	gw := gzip.NewWriter(w)
	tw := tar.NewWriter(gw)

	for _, p := range paths {
		if err := addPath(tw, p); err != nil {
			return constants.ErrCreateArchive.Wrap(err, p)
		}
	}

	if err := tw.Close(); err != nil {
		return constants.ErrCreateArchive.Wrap(err)
	}
	if err := gw.Close(); err != nil {
		return constants.ErrCreateArchive.Wrap(err)
	}
	return nil
}

func addPath(tw *tar.Writer, root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Resolve symlinks for the header
		link := ""
		if info.Mode()&os.ModeSymlink != 0 {
			link, err = os.Readlink(path)
			if err != nil {
				return err
			}
		}

		header, err := tar.FileInfoHeader(info, link)
		if err != nil {
			return err
		}

		header.Name = path

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(tw, f)
		return err
	})
}

// Extract reads a tar.gz archive from r and extracts it into destDir.
// Returns the list of extracted paths.
func Extract(r io.Reader, destDir string) ([]string, error) {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return nil, constants.ErrExtract.Wrap(err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	var extracted []string

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, constants.ErrExtract.Wrap(err)
		}

		// Guard against path traversal
		target := filepath.Join(destDir, header.Name)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)+string(os.PathSeparator)) &&
			filepath.Clean(target) != filepath.Clean(destDir) {
			return nil, constants.ErrExtract.Wrap(nil, fmt.Sprintf("path traversal: %s", header.Name))
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return nil, constants.ErrExtract.Wrap(err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return nil, constants.ErrExtract.Wrap(err)
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return nil, constants.ErrExtract.Wrap(err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return nil, constants.ErrExtract.Wrap(err)
			}
			f.Close()
		}

		extracted = append(extracted, header.Name)
	}

	return extracted, nil
}

// List reads a tar.gz archive from r and returns entry names.
func List(r io.Reader) ([]string, error) {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return nil, constants.ErrExtract.Wrap(err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	var entries []string

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, constants.ErrExtract.Wrap(err)
		}
		entries = append(entries, header.Name)
	}

	return entries, nil
}
