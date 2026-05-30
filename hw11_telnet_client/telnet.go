package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

// TelnetClient определяет интерфейс для операций примитивного TELNET-клиента.
type TelnetClient interface {
	Connect() error
	Send() error
	Receive() error
	io.Closer
}

// telnetClient реализует интерфейс TelnetClient.
type telnetClient struct {
	address string
	timeout time.Duration
	conn    net.Conn
	in      io.ReadCloser
	out     io.Writer
}

// Создаёт новый экземпляр телнет-клиента.
func NewTelnetClient(address string, timeout time.Duration, in io.ReadCloser, out io.Writer) TelnetClient {
	return &telnetClient{
		address: address,
		timeout: timeout,
		in:      in,
		out:     out,
	}
}

// Устанавливает TCP-соединение с целевым адресом.
func (c *telnetClient) Connect() error {
	conn, err := net.DialTimeout("tcp", c.address, c.timeout)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	c.conn = conn
	fmt.Fprintf(os.Stderr, "...Connected to %s\n", c.address)
	return nil
}

// Копирует данные из stdin в сокет.
func (c *telnetClient) Send() error {
	if c.conn == nil {
		return fmt.Errorf("not connected")
	}
	_, err := io.Copy(c.conn, c.in)
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("send error: %w", err)
	}
	return nil
}

// Копирует данные из сокета в stdout.
func (c *telnetClient) Receive() error {
	if c.conn == nil {
		return fmt.Errorf("not connected")
	}
	_, err := io.Copy(c.out, c.conn)
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("receive error: %w", err)
	}
	return nil
}

// Закрывает соединение.
func (c *telnetClient) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}
