package main

import (
	"errors"
	"fmt"
	"io"
	"os"
)

var (
	ErrUnsupportedFile       = errors.New("unsupported file")
	ErrOffsetExceedsFileSize = errors.New("offset exceeds file size")
)

const (
	bufferSize        = 64 * 1024 // 64KB буфер для чтения/записи
	progressThreshold = 0.05      // обновлять прогресс каждые 5%
)

func Copy(fromPath, toPath string, offset, limit int64) error {
	srcFile, err := os.Open(fromPath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	fileInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	if !fileInfo.Mode().IsRegular() {
		return ErrUnsupportedFile
	}

	fileSize := fileInfo.Size()

	if offset > fileSize {
		return ErrOffsetExceedsFileSize
	}

	if offset > 0 {
		_, err = srcFile.Seek(offset, io.SeekStart)
		if err != nil {
			return fmt.Errorf("failed to seek: %w", err)
		}
	}

	bytesToCopy := limit
	if limit == 0 || limit > fileSize-offset {
		bytesToCopy = fileSize - offset
	}

	if bytesToCopy == 0 {
		os.Create(toPath)
		return nil
	}

	dstFile, err := os.Create(toPath)
	if err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}
	defer dstFile.Close()

	limitedReader := io.LimitReader(srcFile, bytesToCopy)
	return copyWithProgress(dstFile, limitedReader, bytesToCopy)
}

func copyWithProgress(dst io.Writer, src io.Reader, total int64) error {
	var (
		copied          int64
		lastUpdateBytes int64
		buf             = make([]byte, bufferSize)
	)

	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, writeErr := dst.Write(buf[:n]); writeErr != nil {
				return fmt.Errorf("failed to write: %w", writeErr)
			}
			copied += int64(n)

			// Обновляем прогресс каждые n% или в конце
			if float64(copied-lastUpdateBytes)/float64(total) >= progressThreshold || copied == total {
				showProgress(copied, total)
				lastUpdateBytes = copied
			}
		}

		if err != nil {
			if err == io.EOF {
				fmt.Printf("\rProgress: 100%% [%d/%d bytes]\n", total, total)
				break
			}
			return fmt.Errorf("failed to read: %w", err)
		}
	}
	return nil
}

func showProgress(copied, total int64) {
	percentage := float64(copied) / float64(total) * 100
	fmt.Printf("\rProgress: %.1f%% [%d/%d bytes]", percentage, copied, total)
}
