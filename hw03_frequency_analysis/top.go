package hw03frequencyanalysis

import (
	"sort"
	"strings"
	"unicode"
)

// Возвращает топ-10 самых частых слов в тексте (базовый вариант).
// Учитывает регистр, знаки препинания являются частью слова.
func Top10(text string) []string {
	if text == "" {
		return []string{}
	}

	// Разбиваем текст на слова по пробельным символам.
	words := strings.Fields(text)

	// Подсчитываем частоту слов
	freq := make(map[string]int)
	for _, word := range words {
		freq[word]++
	}

	return getTopWords(freq, 10)
}

// Возвращает топ-10 самых частых слов (расширенный вариант).
// Не учитывает регистр, удаляет знаки препинания по краям слов.
// Поддерживает Unicode (кириллица, латиница, иероглифы и т.д.).
func Top10Clean(text string) []string {
	if text == "" {
		return []string{}
	}

	rawWords := strings.Fields(text)
	freq := make(map[string]int)

	for _, word := range rawWords {
		cleaned := cleanWord(word)
		if cleaned == "" {
			continue
		}
		// Приводим к нижнему регистру после очистки
		cleaned = strings.ToLower(cleaned)
		freq[cleaned]++
	}

	return getTopWords(freq, 10)
}

// Очищает слово от знаков препинания по краям.
// Если слово состоит только из знаков препинания:
// - длина 1 (например "-") -> возвращается пустая строка (игнорируется)
// - длина > 1 (например "-------") -> возвращается исходное слово.
func cleanWord(word string) string {
	runes := []rune(word)
	start := 0
	end := len(runes)

	// Находим индекс первой буквы или цифры
	for start < end {
		if unicode.IsLetter(runes[start]) || unicode.IsNumber(runes[start]) {
			break
		}
		start++
	}

	// Находим индекс последней буквы или цифры
	for end > start {
		if unicode.IsLetter(runes[end-1]) || unicode.IsNumber(runes[end-1]) {
			break
		}
		end--
	}

	// Если букв/цифр не найдено (слово состоит только из спецсимволов)
	if start >= end {
		// По условию: "-" не является словом, а "-------" является
		if len(runes) == 1 {
			return ""
		}
		// Возвращаем исходное слово как есть (оно будет приведено к LowerCase позже)
		return word
	}

	// Возвращаем очищенную часть слова
	return string(runes[start:end])
}

// Возвращает топ-N слов из частотного словаря.
// Сортировка: по убыванию частоты, при равенстве - лексикографически по возрастанию.
func getTopWords(freq map[string]int, n int) []string {
	if len(freq) == 0 {
		return []string{}
	}

	// Создаем слайс для сортировки
	words := make([]string, 0, len(freq))
	for word := range freq {
		words = append(words, word)
	}

	// Сортируем
	sort.Slice(words, func(i, j int) bool {
		countI := freq[words[i]]
		countJ := freq[words[j]]

		if countI == countJ {
			return words[i] < words[j] // Лексикографически по возрастанию
		}
		return countI > countJ // По частоте по убыванию
	})

	// Возвращаем первые N слов
	if len(words) > n {
		return words[:n]
	}
	return words
}
