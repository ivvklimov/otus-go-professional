package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	from, to      string
	limit, offset int64
)

func init() {
	flag.StringVar(&from, "from", "", "file to read from")
	flag.StringVar(&to, "to", "", "file to write to")
	flag.Int64Var(&limit, "limit", 0, "limit of bytes to copy")
	flag.Int64Var(&offset, "offset", 0, "offset in input file")
}

func main() {
	flag.Parse()

	// Проверяем обязательные параметры
	if from == "" || to == "" {
		fmt.Println("Usage: go-cp -from <file> -to <file> [-offset N] [-limit N]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Выполняем копирование
	err := Copy(from, to, offset, limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
