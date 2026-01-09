package hw02unpackstring

import (
	"errors"
	"strconv"
	"strings"
	"unicode"
)

var ErrInvalidString = errors.New("invalid string")

// Unpack распаковывает строку, содержащую повторяющиеся символы/руны.
// Формат: символ может повторяться указанное количество раз, если после него идет цифра.
// Экранирование через \ разрешено только для цифр и обратного слэша.
func Unpack(s string) (string, error) {
	var result strings.Builder
	runes := []rune(s)
	length := len(runes)

	for i := 0; i < length; i++ {
		// Обработка экранирования
		if runes[i] == '\\' {
			if i == length-1 {
				return "", ErrInvalidString // слэш в конце строки
			}

			i++ // переходим к экранируемому символу
			escapedChar := runes[i]

			// Проверяем, что экранировать можно только цифру или слэш
			if escapedChar != '\\' && !unicode.IsDigit(escapedChar) {
				return "", ErrInvalidString
			}

			// Обрабатываем экранированный символ
			if err := writeChar(&result, runes, &i, length, escapedChar); err != nil {
				return "", err
			}
			continue
		}

		// Проверка: обычная цифра не может идти первой
		if unicode.IsDigit(runes[i]) {
			return "", ErrInvalidString
		}

		// Обработка обычного символа
		char := runes[i]
		if err := writeChar(&result, runes, &i, length, char); err != nil {
			return "", err
		}
	}

	return result.String(), nil
}

// writeChar записывает символ в результат, проверяя следующую цифру для повторения.
func writeChar(builder *strings.Builder, runes []rune, i *int, length int, char rune) error {
	// Проверяем, есть ли следующая цифра
	if *i+1 < length && unicode.IsDigit(runes[*i+1]) {
		// Получаем цифру для повторения
		nextDigit := runes[*i+1]

		// Проверяем, что это не начало числа (не должно быть второй цифры)
		if *i+2 < length && unicode.IsDigit(runes[*i+2]) {
			return ErrInvalidString
		}

		count, _ := strconv.Atoi(string(nextDigit))
		if count > 0 {
			builder.WriteString(strings.Repeat(string(char), count))
		}
		// Пропускаем цифру
		*i++
	} else {
		// Нет цифры после символа - записываем один раз
		builder.WriteRune(char)
	}

	return nil
}
