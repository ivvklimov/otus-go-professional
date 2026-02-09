package main

import (
	"errors"
	"os"
	"os/exec"
	"strings"
)

// RunCmd запускает команду с аргументами (cmd) с переменными окружения из env.
//
//nolint:gosec // false positive: утилита запускает пользовательские команды
func RunCmd(cmd []string, env Environment) int {
	command := exec.Command(cmd[0], cmd[1:]...)

	// Пробрасываем стандартные потоки ввода/вывода/ошибок
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	currentEnv := os.Environ()
	// Создаем новый слайс с предвычисленной емкостью для оптимизации
	newEnv := make([]string, 0, len(currentEnv)+len(env))

	newEnv = append(newEnv, currentEnv...)

	// Применяем модификации из переданного env
	for name, envValue := range env {
		prefix := name + "="

		if envValue.NeedRemove {
			// Удаляем переменную из окружения
			for i := 0; i < len(newEnv); i++ {
				if strings.HasPrefix(newEnv[i], prefix) {
					newEnv = append(newEnv[:i], newEnv[i+1:]...)
				}
			}
		} else {
			// Устанавливаем или обновляем переменную
			found := false
			for i, e := range newEnv {
				if strings.HasPrefix(e, prefix) {
					newEnv[i] = prefix + envValue.Value
					found = true
					break
				}
			}
			if !found {
				newEnv = append(newEnv, prefix+envValue.Value)
			}
		}
	}

	// Устанавливаем подготовленное окружение для команды
	command.Env = newEnv

	if err := command.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode()
		}
		return 1
	}

	return 0
}
