package core

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
)

func (mp *ManifestProcessor) createTarArchive(files []string, tarFile, baseDir string) error {
	mp.logger.V(1).Info("Creating tar archive", "file", tarFile)

	f, err := os.Create(tarFile)
	if err != nil {
		return err
	}
	defer f.Close()

	tw := tar.NewWriter(f)
	defer tw.Close()

	for _, file := range files {
		absFile, err := filepath.Abs(file)
		if err != nil {
			return err
		}
		absBase, err := filepath.Abs(baseDir)
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(absBase, absFile)
		if err != nil {
			return err
		}

		mp.logger.V(1).Info("Adding file to tar", "file", relPath)
		if err := addToTar(tw, file, relPath); err != nil {
			return err
		}
	}

	return nil
}

func addToTar(tw *tar.Writer, filename, relPath string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(info, info.Name())
	if err != nil {
		return err
	}

	header.Name = relPath

	err = tw.WriteHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(tw, file)
	if err != nil {
		return err
	}

	return nil
}

func (mp *ManifestProcessor) extractTarArchive(tarFile, destDir string) error {
	mp.logger.V(1).Info("Extracting tar archive", "from", tarFile, "to", destDir)

	f, err := os.Open(tarFile)
	if err != nil {
		return err
	}
	defer f.Close()

	tr := tar.NewReader(f)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(destDir, header.Name)
		mp.logger.V(1).Info("Extracting file", "file", target)

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}

	return nil
}
