package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadDir(t *testing.T) {
	t.Run("valid directory", func(t *testing.T) {
		env, err := ReadDir("testdata/env")
		require.NoError(t, err)
		require.Len(t, env, 5)

		require.Equal(t, "bar", env["BAR"].Value)
		require.False(t, env["BAR"].NeedRemove)

		require.Equal(t, "   foo\nwith new line", env["FOO"].Value)
		require.False(t, env["FOO"].NeedRemove)

		require.Equal(t, "\"hello\"", env["HELLO"].Value)
		require.False(t, env["HELLO"].NeedRemove)

		require.Equal(t, "", env["EMPTY"].Value)
		require.False(t, env["EMPTY"].NeedRemove)

		require.True(t, env["UNSET"].NeedRemove)
	})

	t.Run("invalid directory", func(t *testing.T) {
		_, err := ReadDir("nonexistent")
		require.Error(t, err)
	})

	t.Run("file with equals sign in name", func(t *testing.T) {
		tmpDir := t.TempDir()
		err := os.WriteFile(filepath.Join(tmpDir, "INVALID=NAME"), []byte("value"), 0o644)
		require.NoError(t, err)

		env, err := ReadDir(tmpDir)
		require.NoError(t, err)
		require.Empty(t, env)
	})

	t.Run("empty file", func(t *testing.T) {
		tmpDir := t.TempDir()
		err := os.WriteFile(filepath.Join(tmpDir, "EMPTYFILE"), []byte{}, 0o644)
		require.NoError(t, err)

		env, err := ReadDir(tmpDir)
		require.NoError(t, err)
		require.True(t, env["EMPTYFILE"].NeedRemove)
	})

	t.Run("file with null bytes", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Записываем без завершающего null байта
		content := []byte("value\x00with\x00nulls")
		err := os.WriteFile(filepath.Join(tmpDir, "WITHNULLS"), content, 0o644)
		require.NoError(t, err)

		env, err := ReadDir(tmpDir)
		require.NoError(t, err)
		require.Equal(t, "value\nwith\nnulls", env["WITHNULLS"].Value)
	})

	t.Run("file with null bytes and newline", func(t *testing.T) {
		tmpDir := t.TempDir()
		// С завершающим null и переводом строки
		content := []byte("value\x00with\x00nulls\nnext line")
		err := os.WriteFile(filepath.Join(tmpDir, "WITHNULLSNL"), content, 0o644)
		require.NoError(t, err)

		env, err := ReadDir(tmpDir)
		require.NoError(t, err)
		// Только первая строка, null заменены на \n
		require.Equal(t, "value\nwith\nnulls", env["WITHNULLSNL"].Value)
	})

	t.Run("file with trailing spaces", func(t *testing.T) {
		tmpDir := t.TempDir()
		content := []byte("value   \t  ")
		err := os.WriteFile(filepath.Join(tmpDir, "TRAILING"), content, 0o644)
		require.NoError(t, err)

		env, err := ReadDir(tmpDir)
		require.NoError(t, err)
		require.Equal(t, "value", env["TRAILING"].Value)
	})

	t.Run("file with only newline", func(t *testing.T) {
		tmpDir := t.TempDir()
		content := []byte("\n")
		err := os.WriteFile(filepath.Join(tmpDir, "NEWLINE"), content, 0o644)
		require.NoError(t, err)

		env, err := ReadDir(tmpDir)
		require.NoError(t, err)
		require.Equal(t, "", env["NEWLINE"].Value)
		require.False(t, env["NEWLINE"].NeedRemove)
	})

	t.Run("file with windows newline", func(t *testing.T) {
		tmpDir := t.TempDir()
		content := []byte("value\r\n")
		err := os.WriteFile(filepath.Join(tmpDir, "WINNL"), content, 0o644)
		require.NoError(t, err)

		env, err := ReadDir(tmpDir)
		require.NoError(t, err)
		require.Equal(t, "value", env["WINNL"].Value)
	})
}
