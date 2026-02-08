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

const bufferSize = 64 * 1024 // 64KB буфер для чтения/записи

func Copy(fromPath, toPath string, offset, limit int64) error {
	// Открываем исходный файл
	srcFile, err := os.Open(fromPath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Получаем информацию о файле для проверки размера
	fileInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	fileSize := fileInfo.Size()
	if fileSize == 0 {
		return ErrUnsupportedFile
	}

	// Проверяем, что offset не превышает размер файла
	if offset > fileSize {
		return ErrOffsetExceedsFileSize
	}

	// Перемещаем указатель в исходном файле
	if offset > 0 {
		_, err = srcFile.Seek(offset, io.SeekStart)
		if err != nil {
			return fmt.Errorf("failed to seek: %w", err)
		}
	}

	// Рассчитываем количество байт для копирования
	bytesToCopy := limit
	if limit == 0 || limit > fileSize-offset {
		bytesToCopy = fileSize - offset
	}

	// Если нечего копировать, создаем пустой файл
	if bytesToCopy == 0 {
		os.Create(toPath)
		return nil
	}

	// Создаем файл назначения
	dstFile, err := os.Create(toPath)
	if err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}
	defer dstFile.Close()

	// Копируем с ограничением по размеру и отображением прогресса
	limitedReader := io.LimitReader(srcFile, bytesToCopy)
	return copyWithProgress(dstFile, limitedReader, bytesToCopy)
}

// copyWithProgress копирует данные с отображением прогресса в процентах.
func copyWithProgress(dst io.Writer, src io.Reader, total int64) error {
	var copied int64
	buf := make([]byte, bufferSize)

	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, writeErr := dst.Write(buf[:n]); writeErr != nil {
				return fmt.Errorf("failed to write: %w", writeErr)
			}
			copied += int64(n)
			showProgress(copied, total)
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

// showProgress выводит прогресс копирования в консоль.
func showProgress(copied, total int64) {
	percentage := float64(copied) / float64(total) * 100
	fmt.Printf("\rProgress: %.1f%% [%d/%d bytes]", percentage, copied, total)
}
