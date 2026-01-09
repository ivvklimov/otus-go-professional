package hw03frequencyanalysis

import (
	"regexp"
	"sort"
	"strings"
)

var cleanWordRegex = regexp.MustCompile(`^[^a-zA-Zа-яА-Я0-9]+|[^a-zA-Zа-яА-Я0-9]+$`)

// Top10 возвращает топ-10 самых частых слов в тексте.
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

// Top10Clean возвращает топ-10 с учетом регистра и удалением знаков препинания.
func Top10Clean(text string) []string {
	if text == "" {
		return []string{}
	}

	// Разбиваем текст на слова по пробельным символам
	rawWords := strings.Fields(text)
	freq := make(map[string]int)

	for _, word := range rawWords {
		// Очищаем слово от знаков препинания по краям
		cleaned := cleanWordRegex.ReplaceAllString(word, "")

		// Если после очистки ничего не осталось, пропускаем
		if cleaned == "" {
			continue
		}

		// Приводим к нижнему регистру
		cleaned = strings.ToLower(cleaned)
		freq[cleaned]++
	}

	return getTopWords(freq, 10)
}

// getTopWords возвращает топ-N слов из частотного словаря.
func getTopWords(freq map[string]int, n int) []string {
	if len(freq) == 0 {
		return []string{}
	}

	// Создаем слайс для сортировки
	words := make([]string, 0, len(freq))
	for word := range freq {
		words = append(words, word)
	}

	// Сортируем по частоте (убывание), затем лексикографически
	sort.Slice(words, func(i, j int) bool {
		if freq[words[i]] == freq[words[j]] {
			return words[i] < words[j]
		}
		return freq[words[i]] > freq[words[j]]
	})

	// Возвращаем первые N слов
	if len(words) > n {
		return words[:n]
	}
	return words
}
