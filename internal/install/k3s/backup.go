package k3s

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type backedupFile struct {
	Filename string
	Filemode os.FileMode
	Backup   string
}

func backupK3S() ([]backedupFile, error) {
	tmp, err := os.MkdirTemp("/tmp", "homecli_backup")
	if err != nil {
		return nil, fmt.Errorf("mktemp: %w", err)
	}

	matches, err := filepath.Glob(filepath.Join(k3sImagesPath, "k3s-airgap-images-*.tar.gz"))
	if err != nil {
		return nil, fmt.Errorf("airgap images: %w", err)
	}
	matches = append(matches, k3sBinary())

	logger.Info().Interface("files", matches).Msgf("Backing up files to %q", tmp)

	var results = make([]backedupFile, 0, len(matches))

	for _, fname := range matches {
		result, err := backupFile(tmp, fname)
		if err != nil {
			logger.Error().Err(err).Msg("Backup failed")
			return results, err
		}
		results = append(results, result)
	}

	logger.Info().Msg("Backup done")

	return results, err
}

func backupFile(tmp string, fname string) (backedupFile, error) {
	var result = backedupFile{
		Filename: fname,
	}

	stat, err := os.Stat(fname)
	if err != nil {
		return result, fmt.Errorf("stat %q: %w", fname, err)
	}

	result.Filemode = stat.Mode()

	file, err := os.Open(fname)
	if err != nil {
		return result, fmt.Errorf("open %q: %w", fname, err)
	}
	defer file.Close()

	result.Backup = filepath.Join(tmp, filepath.Base(fname))

	tmpfile, err := os.OpenFile(result.Backup, os.O_CREATE|os.O_WRONLY, stat.Mode())
	if err != nil {
		return result, fmt.Errorf("open tmpfile: %w", err)
	}
	defer tmpfile.Close()

	_, err = io.Copy(tmpfile, file)
	if err != nil {
		return result, fmt.Errorf("copy: %w", err)
	}

	return result, nil
}

func restore(files []backedupFile) error {
	logger.Info().Msg("Restoring from backup")

	for _, file := range files {
		if err := restoreFile(file); err != nil {
			logger.Error().
				Err(err).Interface("files", files).
				Msg("Restoring from backup failed due to error, please copy files manually and restart the service")
			return err
		}
	}

	logger.Info().Msg("Original files was restored")
	return nil
}

func restoreFile(file backedupFile) error {
	tmpfile, err := os.Open(file.Backup)
	if err != nil {
		return fmt.Errorf("open %q: %w", file.Backup, err)
	}
	defer tmpfile.Close()

	restored, err := os.OpenFile(file.Filename, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, file.Filemode)
	if err != nil {
		return fmt.Errorf("open %q: %w", file.Filename, err)
	}
	defer restored.Close()

	_, err = io.Copy(restored, tmpfile)
	if err != nil {
		return fmt.Errorf("copy: %w", err)
	}
	return nil
}
