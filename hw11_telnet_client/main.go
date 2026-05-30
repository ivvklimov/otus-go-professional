package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var timeout time.Duration
	flag.DurationVar(&timeout, "timeout", 10*time.Second, "connection timeout")
	flag.Parse()

	if flag.NArg() != 2 {
		return fmt.Errorf("usage: go-telnet [--timeout=duration] host port")
	}
	host, port := flag.Arg(0), flag.Arg(1)
	address := net.JoinHostPort(host, port)

	// Создаём контекст с обработкой сигналов (SIGINT, SIGTERM)
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	client := NewTelnetClient(address, timeout, os.Stdin, os.Stdout)

	if err := client.Connect(); err != nil {
		return fmt.Errorf("connection error: %w", err)
	}
	defer client.Close()

	// Каналы для сигналов о завершении горутин
	sendDone := make(chan error, 1)
	recvDone := make(chan error, 1)

	// Горутина: отправка данных из stdin в сокет
	go func() {
		sendDone <- client.Send()
	}()

	// Горутина: получение данных из сокета в stdout
	go func() {
		recvDone <- client.Receive()
	}()

	// Ждём завершения любой горутины или отмены контекста
	select {
	case <-ctx.Done():
		// Получен сигнал SIGINT/SIGTERM
		return nil

	case err := <-sendDone:
		if err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("send: %w", err)
		}
		client.Close()
		<-recvDone
		fmt.Fprintln(os.Stderr, "...EOF")
		return nil

	case err := <-recvDone:
		// Горутина Receive завершилась (сервер закрыл соединение)
		if err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("receive: %w", err)
		}
		// Закрываем соединение, чтобы разблокировать Send(), если он ещё ждёт stdin
		client.Close()
		fmt.Fprintln(os.Stderr, "...Connection was closed by peer")
		select {
		case <-sendDone:
		case <-time.After(100 * time.Millisecond):
		}
		return nil
	}
}
