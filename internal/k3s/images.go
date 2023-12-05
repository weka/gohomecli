package k3s

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
)

func unzippedData(imagePath string) ([]byte, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		return nil, err
	}

	_, err = file.Seek(0, 0)
	if err != nil {
		return nil, err
	}

	mime := http.DetectContentType(buffer)
	fmt.Println(mime)
	if mime == "application/x-gzip" {
		reader, err := gzip.NewReader(file)
		if err != nil {
			return nil, err
		}

		return io.ReadAll(reader)
	} else if mime == "application/x-tar" {
		return io.ReadAll(file)
	}

	return nil, fmt.Errorf("unknown file type (%s)", mime)
}

func ImportDockerImages(imagePaths []string, failFast bool) error {
	var importErrors []error
	for _, imagePath := range imagePaths {
		data, err := unzippedData(imagePath)
		if err != nil {
			logger.Warn().
				Err(err).
				Str("imagePath", imagePath).
				Msg("Failed to read image data")

			if failFast {
				return err
			} else {
				importErrors = append(importErrors, err)
				continue
			}
		}

		reader := bytes.NewBuffer(data)
		cmd := exec.Command("ctr", "image", "import", "--digests", "sha256", "--", "-")
		cmd.Stdin = reader

		// stderr, err := cmd.StderrPipe()
		// if err != nil {
		// 	logger.Warn().
		// 		Err(err).
		// 		Str("imagePath", imagePath).
		// 		Msg("Failed to capture import command output")

		// 	if failFast {
		// 		return err
		// 	} else {
		// 		importErrors = append(importErrors, err)
		// 		continue
		// 	}
		// }

		// stdout, err := cmd.StdoutPipe()
		// if err != nil {
		// 	logger.Warn().
		// 		Err(err).
		// 		Str("imagePath", imagePath).
		// 		Msg("Failed to capture import command output")

		// 	if failFast {
		// 		return err
		// 	} else {
		// 		importErrors = append(importErrors, err)
		// 		continue
		// 	}
		// }

		err = cmd.Start()
		if err != nil {
			logger.Warn().
				Err(err).
				Str("imagePath", imagePath).
				Msg("Failed run import command")

			if failFast {
				return err
			} else {
				importErrors = append(importErrors, err)
				continue
			}
		}

		// go io.Copy(utils.NewWriteScanner(k3sLogParser(utils.InfoLevel)), stdout)
		// go io.Copy(utils.NewWriteScanner(k3sLogParser(utils.InfoLevel)), stderr)

		err = cmd.Wait()
		if err != nil {
			logger.Warn().
				Err(err).
				Str("imagePath", imagePath).
				Msg("Failed to import image")

			if failFast {
				return err
			} else {
				importErrors = append(importErrors, err)
				continue
			}
		}
	}

	if len(importErrors) > 0 {
		return errors.Join(importErrors...)
	}

	return nil
}
