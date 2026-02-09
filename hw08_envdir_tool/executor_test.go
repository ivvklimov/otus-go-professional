package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunCmd(t *testing.T) {
	t.Run("set and remove variables", func(t *testing.T) {
		// Создаем временный скрипт
		tmpDir := t.TempDir()
		scriptPath := filepath.Join(tmpDir, "test.sh")
		script := `#!/bin/sh
echo "NEWVAR=$NEWVAR"
echo "TOREMOVE=$TOREMOVE"
echo "EXISTING=$EXISTING"
echo "EMPTYVAR=$EMPTYVAR"
`
		err := os.WriteFile(scriptPath, []byte(script), 0o755)
		require.NoError(t, err)

		// Используем второй вариант RunCmd (без модификации глобального окружения)
		env := Environment{
			"NEWVAR":   EnvValue{Value: "newvalue", NeedRemove: false},
			"TOREMOVE": EnvValue{NeedRemove: true},
			"EXISTING": EnvValue{Value: "updated", NeedRemove: false},
			"EMPTYVAR": EnvValue{Value: "", NeedRemove: false},
		}

		// Захватываем вывод
		var buf bytes.Buffer
		cmd := exec.Command(scriptPath)
		cmd.Env = buildEnvironment(env)
		cmd.Stdout = &buf
		cmd.Stderr = &buf

		err = cmd.Run()
		require.NoError(t, err)

		expected := `NEWVAR=newvalue
TOREMOVE=
EXISTING=updated
EMPTYVAR=
`
		require.Equal(t, expected, buf.String())
	})

	t.Run("command not found", func(t *testing.T) {
		env := Environment{}
		code := RunCmd([]string{"nonexistentcommand12345"}, env)
		require.Equal(t, 1, code)
	})

	t.Run("command returns exit code", func(t *testing.T) {
		tmpDir := t.TempDir()
		scriptPath := filepath.Join(tmpDir, "test.sh")
		script := `#!/bin/sh
exit 42
`
		err := os.WriteFile(scriptPath, []byte(script), 0o755)
		require.NoError(t, err)

		env := Environment{}
		code := RunCmd([]string{scriptPath}, env)
		require.Equal(t, 42, code)
	})
}

func TestIntegration(t *testing.T) {
	buildCmd := exec.Command("go", "build", "-o", "go-envdir", ".")
	err := buildCmd.Run()
	require.NoError(t, err)
	defer os.Remove("go-envdir")

	os.Setenv("HELLO", "SHOULD_REPLACE")
	os.Setenv("FOO", "SHOULD_REPLACE")
	os.Setenv("UNSET", "SHOULD_REMOVE")
	os.Setenv("ADDED", "from original env")
	os.Setenv("EMPTY", "SHOULD_BE_EMPTY")

	// Запуск интеграционного теста
	cmd := exec.Command(
		"./go-envdir",
		"testdata/env",
		"/bin/bash",
		"testdata/echo.sh",
		"arg1=1",
		"arg2=2",
	)

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err = cmd.Run()
	require.NoError(t, err)

	// Правильный ожидаемый вывод из test.sh
	expected := `HELLO is ("hello")
BAR is (bar)
FOO is (   foo
with new line)
UNSET is ()
ADDED is (from original env)
EMPTY is ()
arguments are arg1=1 arg2=2
`
	require.Equal(t, expected, buf.String())
}

// Вспомогательная функция из второго варианта RunCmd.
func buildEnvironment(env Environment) []string {
	result := os.Environ()

	for name, envValue := range env {
		if envValue.NeedRemove {
			result = removeEnvVar(result, name)
		} else {
			result = setEnvVar(result, name, envValue.Value)
		}
	}

	return result
}

func removeEnvVar(env []string, name string) []string {
	result := make([]string, 0, len(env))
	for _, e := range env {
		if !strings.HasPrefix(e, name+"=") {
			result = append(result, e)
		}
	}
	return result
}

func setEnvVar(env []string, name, value string) []string {
	prefix := name + "="
	for i, e := range env {
		if strings.HasPrefix(e, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}
