package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCopy(t *testing.T) {
	inputPath := filepath.Join("testdata", "input.txt")

	tests := []struct {
		name         string
		offset       int64
		limit        int64
		expectedFile string
		expectedSize int64
		expectError  bool
		errorType    error
	}{
		{
			name:         "copy entire file",
			offset:       0,
			limit:        0,
			expectedFile: "out_offset0_limit0.txt",
			expectedSize: 6617,
		},
		{
			name:         "copy with limit 10",
			offset:       0,
			limit:        10,
			expectedFile: "out_offset0_limit10.txt",
			expectedSize: 10,
		},
		{
			name:         "copy with limit 1000",
			offset:       0,
			limit:        1000,
			expectedFile: "out_offset0_limit1000.txt",
			expectedSize: 1000,
		},
		{
			name:         "copy with limit 10000",
			offset:       0,
			limit:        10000,
			expectedFile: "out_offset0_limit10000.txt",
			expectedSize: 6617,
		},
		{
			name:         "copy with offset 100 and limit 1000",
			offset:       100,
			limit:        1000,
			expectedFile: "out_offset100_limit1000.txt",
			expectedSize: 1000,
		},
		{
			name:         "copy with offset 6000 and limit 1000",
			offset:       6000,
			limit:        1000,
			expectedFile: "out_offset6000_limit1000.txt",
			expectedSize: 617,
		},
		{
			name:        "offset exceeds file size",
			offset:      10000,
			limit:       100,
			expectError: true,
			errorType:   ErrOffsetExceedsFileSize,
		},
		{
			name:         "offset at the end of file",
			offset:       6617,
			limit:        100,
			expectedSize: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			outputPath := filepath.Join(t.TempDir(), "output.txt")
			err := Copy(inputPath, outputPath, tc.offset, tc.limit)

			if tc.expectError {
				assertError(t, err, tc.errorType)
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			assertCopyResult(t, outputPath, tc.expectedFile, tc.expectedSize)
		})
	}
}

func assertError(t *testing.T, err error, expected error) {
	t.Helper()
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
	if expected != nil && !errors.Is(err, expected) {
		t.Errorf("expected error type %v, got %v", expected, err)
	}
}

func assertCopyResult(t *testing.T, outputPath, expectedFile string, expectedSize int64) {
	t.Helper()

	if expectedFile != "" {
		expectedPath := filepath.Join("testdata", expectedFile)
		expectedData, err := os.ReadFile(expectedPath)
		if err != nil {
			t.Fatalf("failed to read expected file: %v", err)
		}

		actualData, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("failed to read actual file: %v", err)
		}

		if !bytes.Equal(expectedData, actualData) {
			t.Errorf("file content mismatch")
		}
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("failed to stat output file: %v", err)
	}
	if expectedSize >= 0 && info.Size() != expectedSize {
		t.Errorf("expected file size %d, got %d", expectedSize, info.Size())
	}
}

func TestCopyInvalidInput(t *testing.T) {
	t.Run("non-existent source file", func(t *testing.T) {
		outputPath := filepath.Join(t.TempDir(), "output.txt")
		err := Copy("/non/existent/file.txt", outputPath, 0, 100)
		if err == nil {
			t.Error("expected error for non-existent file")
		}
	})

	t.Run("empty source file", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "empty.txt")
		if err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()
		defer os.Remove(tmpFile.Name())

		outputPath := filepath.Join(t.TempDir(), "output.txt")
		err = Copy(tmpFile.Name(), outputPath, 0, 100)
		if !errors.Is(err, ErrUnsupportedFile) {
			t.Errorf("expected ErrUnsupportedFile, got %v", err)
		}
	})
}
