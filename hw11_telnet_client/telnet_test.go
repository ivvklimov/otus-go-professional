package main

import (
	"bytes"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTelnetClient(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:")
		require.NoError(t, err)
		defer func() { require.NoError(t, l.Close()) }()

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()

			in := &bytes.Buffer{}
			out := &bytes.Buffer{}

			timeout, err := time.ParseDuration("10s")
			require.NoError(t, err)

			client := NewTelnetClient(l.Addr().String(), timeout, io.NopCloser(in), out)
			require.NoError(t, client.Connect())
			defer func() { require.NoError(t, client.Close()) }()

			in.WriteString("hello\n")
			err = client.Send()
			require.NoError(t, err)

			err = client.Receive()
			require.NoError(t, err)
			require.Equal(t, "world\n", out.String())
		}()

		go func() {
			defer wg.Done()

			conn, err := l.Accept()
			require.NoError(t, err)
			require.NotNil(t, conn)
			defer func() { require.NoError(t, conn.Close()) }()

			request := make([]byte, 1024)
			n, err := conn.Read(request)
			require.NoError(t, err)
			require.Equal(t, "hello\n", string(request)[:n])

			n, err = conn.Write([]byte("world\n"))
			require.NoError(t, err)
			require.NotEqual(t, 0, n)
		}()

		wg.Wait()
	})
}

// Ошибка подключения к закрытому порту (connection refused).
func TestTelnetClient_ConnectionRefused(t *testing.T) {
	in := &bytes.Buffer{}
	out := &bytes.Buffer{}

	client := NewTelnetClient("127.0.0.1:1", 100*time.Millisecond, io.NopCloser(in), out)
	err := client.Connect()
	require.Error(t, err)
}

// Таймаут при подключении к недоступному хосту.
func TestTelnetClient_Timeout(t *testing.T) {
	in := &bytes.Buffer{}
	out := &bytes.Buffer{}

	// Непубличный IP-адрес, который не ответит
	client := NewTelnetClient("192.0.2.1:80", 500*time.Millisecond, io.NopCloser(in), out)
	err := client.Connect()
	require.Error(t, err)
}

// Завершение при EOF от stdin (аналог Ctrl+D).
func TestTelnetClient_EOF(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:")
	require.NoError(t, err)
	defer func() { require.NoError(t, l.Close()) }()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		in := bytes.NewBufferString("test\n")
		out := &bytes.Buffer{}

		client := NewTelnetClient(l.Addr().String(), 10*time.Second, io.NopCloser(in), out)
		require.NoError(t, client.Connect())
		defer func() { require.NoError(t, client.Close()) }()

		err = client.Send()
		require.NoError(t, err)
	}()

	go func() {
		defer wg.Done()

		conn, err := l.Accept()
		require.NoError(t, err)
		defer func() { require.NoError(t, conn.Close()) }()

		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		require.NoError(t, err)
		require.Equal(t, "test\n", string(buf)[:n])
	}()

	wg.Wait()
}

// Сервер закрывает соединение, клиент должен это обнаружить.
func TestTelnetClient_ServerCloses(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:")
	require.NoError(t, err)
	defer func() { require.NoError(t, l.Close()) }()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		in := &bytes.Buffer{}
		out := &bytes.Buffer{}

		client := NewTelnetClient(l.Addr().String(), 10*time.Second, io.NopCloser(in), out)
		require.NoError(t, client.Connect())
		defer func() { require.NoError(t, client.Close()) }()

		// Сначала получаем данные от сервера
		err = client.Receive()
		require.NoError(t, err)
		require.Equal(t, "from server\n", out.String())
	}()

	go func() {
		defer wg.Done()

		conn, err := l.Accept()
		require.NoError(t, err)

		// Отправляем данные и закрываем соединение
		_, err = conn.Write([]byte("from server\n"))
		require.NoError(t, err)
		require.NoError(t, conn.Close())
	}()

	wg.Wait()
}

// Одновременная отправка и получение (несколько сообщений).
func TestTelnetClient_Concurrent(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:")
	require.NoError(t, err)
	defer func() { require.NoError(t, l.Close()) }()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		in := bytes.NewBufferString("msg1\nmsg2\n")
		out := &bytes.Buffer{}

		client := NewTelnetClient(l.Addr().String(), 10*time.Second, io.NopCloser(in), out)
		require.NoError(t, client.Connect())
		defer func() { require.NoError(t, client.Close()) }()

		// Запускаем Send и Receive параллельно
		var work sync.WaitGroup
		work.Add(2)

		go func() {
			defer work.Done()
			require.NoError(t, client.Send())
		}()

		go func() {
			defer work.Done()
			require.NoError(t, client.Receive())
		}()

		work.Wait()
		require.Contains(t, out.String(), "reply1\n")
	}()

	go func() {
		defer wg.Done()

		conn, err := l.Accept()
		require.NoError(t, err)
		defer func() { require.NoError(t, conn.Close()) }()

		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		require.NoError(t, err)
		require.Contains(t, string(buf)[:n], "msg1\n")

		_, err = conn.Write([]byte("reply1\n"))
		require.NoError(t, err)
	}()

	wg.Wait()
}
