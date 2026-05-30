package hw10programoptimization

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"
)

// Тип для хранения статистики по доменам.
type DomainStat map[string]int

// Минимальная структура для парсинга только нужного поля Email.
type user struct {
	Email string `json:"Email"` //nolint:tagliatelle // входной JSON использует "Email" с заглавной буквы
}

// Подсчитывает количество email-доменов, оканчивающихся на указанный домен первого уровня.
// Реализация оптимизирована для обработки больших объёмов данных:
//   - потоковое чтение построчно (bufio.Scanner) — без загрузки всего файла в память;
//   - парсинг только поля Email — минимизация накладных расходов на JSON;
//   - замена регулярных выражений на строковые операции — ускорение фильтрации;
//   - однократное приведение к нижнему регистру и извлечение домена — снижение аллокаций.
func GetDomainStat(r io.Reader, domain string) (DomainStat, error) {
	result := make(DomainStat)
	domainSuffix := "." + domain

	scanner := bufio.NewScanner(r)
	// Увеличиваем буфер на случай длинных строк (стандартный ~64KB может быть недостаточен).
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var u user
		if err := json.Unmarshal(line, &u); err != nil {
			return nil, err
		}

		if u.Email == "" {
			continue
		}

		// Находим позицию @ для извлечения доменной части.
		atIdx := strings.IndexByte(u.Email, '@')
		if atIdx == -1 || atIdx+1 >= len(u.Email) {
			continue
		}

		// Нормализуем домен - приводим к нижнему регистру один раз.
		emailDomain := strings.ToLower(u.Email[atIdx+1:])

		// Проверяем, что домен оканчивается на нужный TLD (например, .gov, .biz).
		if !strings.HasSuffix(emailDomain, domainSuffix) {
			continue
		}

		result[emailDomain]++
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
