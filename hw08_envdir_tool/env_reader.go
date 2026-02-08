package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Environment map[string]EnvValue

// EnvValue помогает различить пустые файлы и файлы с первой пустой строкой.
type EnvValue struct {
	Value      string
	NeedRemove bool
}

// ReadDir читает указанную директорию и возвращает карту переменных окружения.
// Переменные представлены как файлы, где имя файла - имя переменной, первая строка файла - значение.
func ReadDir(dir string) (Environment, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	env := make(Environment)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.Contains(name, "=") {
			continue
		}

		filePath := filepath.Join(dir, name)
		fileInfo, err := entry.Info()
		if err != nil {
			return nil, err
		}

		if fileInfo.Size() == 0 {
			env[name] = EnvValue{NeedRemove: true}
			continue
		}

		file, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}

		content, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			return nil, err
		}

		value := processFirstLine(content)
		env[name] = EnvValue{Value: value, NeedRemove: false}
	}

	return env, nil
}

func processFirstLine(content []byte) string {
	// Находим первый перенос строки
	idx := bytes.IndexByte(content, '\n')
	if idx == -1 {
		idx = len(content)
	}

	firstLine := content[:idx]

	// Заменяем 0x00 на \n
	firstLine = bytes.ReplaceAll(firstLine, []byte{0}, []byte("\n"))

	// Удаляем пробелы и табуляцию в конце
	result := strings.TrimRight(string(firstLine), " \t")

	// Также удаляем \r (Windows стиль окончания строк)
	result = strings.TrimRight(result, "\r")

	return result
}
