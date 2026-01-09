package hw02unpackstring

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnpack(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{input: "a4bc2d5e", expected: "aaaabccddddde"},
		{input: "abccd", expected: "abccd"},
		{input: "", expected: ""},
		{input: "aaa0b", expected: "aab"},
		{input: "🙃0", expected: ""},
		{input: "aaф0b", expected: "aab"},
		// uncomment if task with asterisk completed
		{input: `qwe\4\5`, expected: `qwe45`},
		{input: `qwe\45`, expected: `qwe44444`},
		{input: `qwe\\5`, expected: `qwe\\\\\`},
		{input: `qwe\\\3`, expected: `qwe\3`},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result, err := Unpack(tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestUnpackInvalidString(t *testing.T) {
	invalidStrings := []string{"3abc", "45", "aaa10b"}
	for _, tc := range invalidStrings {
		t.Run(tc, func(t *testing.T) {
			_, err := Unpack(tc)
			require.Truef(t, errors.Is(err, ErrInvalidString), "actual error %q", err)
		})
	}
}

func TestUnpackAdditionalCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		// Базовые случаи с разными символами
		{name: "single_repeated_char", input: "a5", expected: "aaaaa", hasError: false},
		{name: "multiple_single_chars", input: "abc", expected: "abc", hasError: false},
		{name: "mixed_repetitions", input: "a1b2c3", expected: "abbccc", hasError: false},
		{name: "zero_repetition_middle", input: "ab0c", expected: "ac", hasError: false},
		{name: "zero_repetition_end", input: "abc0", expected: "ab", hasError: false},
		{name: "all_zeros", input: "a0b0c0", expected: "", hasError: false},

		// Специальные символы и Unicode
		{name: "unicode_emoji_repeat", input: "😀3", expected: "😀😀😀", hasError: false},
		{name: "unicode_cyrillic", input: "а2б2в2", expected: "ааббвв", hasError: false},
		{name: "unicode_mixed", input: "a1ß2😀1", expected: "aßß😀", hasError: false},
		{name: "newline_char", input: "a\n2b", expected: "a\n\nb", hasError: false},
		{name: "tab_char", input: "a\t2b", expected: "a\t\tb", hasError: false},

		// Граничные случаи
		{name: "single_char", input: "a", expected: "a", hasError: false},
		{name: "single_char_with_one", input: "a1", expected: "a", hasError: false},
		{name: "empty_string", input: "", expected: "", hasError: false},
		{name: "only_zero_repetitions", input: "a0", expected: "", hasError: false},

		// Случаи с ошибками (должны возвращать ErrInvalidString)
		{name: "starts_with_digit", input: "5abc", expected: "", hasError: true},
		{name: "only_digits", input: "123", expected: "", hasError: true},
		{name: "multiple_digits_number", input: "a10b", expected: "", hasError: true},
		{name: "digit_after_digit", input: "a12", expected: "", hasError: true},
		{name: "three_digit_number", input: "a100", expected: "", hasError: true},
		{name: "multiple_zeros_before_char", input: "a00b", expected: "", hasError: true},
		{name: "just_digit", input: "5", expected: "", hasError: true},
		{name: "zero_first", input: "0a", expected: "", hasError: true},

		{name: "digit_at_end_repeats_last_char", input: "abc5", expected: "abccccc", hasError: false},
		{name: "multiple_digits_at_end", input: "a2b3c4d5", expected: "aabbbccccddddd", hasError: false},
		{name: "single_digit_at_end", input: "x5", expected: "xxxxx", hasError: false},
		{name: "zero_at_end", input: "abc0", expected: "ab", hasError: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Unpack(tc.input)

			if tc.hasError {
				require.Error(t, err)
				require.Truef(t, errors.Is(err, ErrInvalidString),
					"expected ErrInvalidString, got %v", err)
				require.Empty(t, result)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestUnpackEscapeSequences(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		// Валидные escape-последовательности
		{name: "escape_digit", input: `\3`, expected: "3"},
		{name: "escape_digit_repeat", input: `\32`, expected: "33"},
		{name: "escape_slash", input: `\\`, expected: `\`},
		{name: "escape_slash_repeat", input: `\\2`, expected: `\\`},
		{name: "escape_multiple_digits", input: `\1\2\3`, expected: "123"},
		{name: "mixed_escape", input: `a\1b\2c`, expected: "a1b2c"},
		{name: "escape_with_repetition", input: `a\12`, expected: "a11"},
		{name: "escape_zero", input: `\0`, expected: "0"},
		{name: "escape_zero_repeat", input: `\02`, expected: "00"},
		{name: "escape_nine", input: `\9`, expected: "9"},

		// Некорректные escape-последовательности
		{name: "escape_at_end", input: `abc\`, expected: "", hasError: true},
		{name: "invalid_escape_char", input: `\a`, expected: "", hasError: true},
		{name: "escape_newline", input: `\n`, expected: "", hasError: true},
		{name: "escape_tab", input: `\t`, expected: "", hasError: true},
		{name: "escape_space", input: `\ `, expected: "", hasError: true},
		{name: "escape_letter", input: `\x`, expected: "", hasError: true},

		// Комплексные случаи с экранированием
		{name: "escape_digit_sequence", input: `\4\5\6`, expected: "456"},
		{name: "escape_in_middle", input: `ab\3cd`, expected: "ab3cd"},
		{name: "escape_with_following_digit", input: `a\23`, expected: "a222"},
		{name: "multiple_escapes_with_repetition", input: `\1\22`, expected: "122"},
		{name: "escape_slash_with_digit_after", input: `\\3`, expected: `\\\`},
		{name: "escape_digit_after_slash", input: `a\\2b`, expected: `a\\b`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Unpack(tc.input)

			if tc.hasError {
				require.Error(t, err)
				require.Truef(t, errors.Is(err, ErrInvalidString),
					"expected ErrInvalidString, got %v", err)
				require.Empty(t, result)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestUnpackEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		// Краевые случаи
		{name: "back_to_back_digits_error_case", input: "a23", expected: "", hasError: true},
		{name: "zero_after_escape", input: `a\0`, expected: "a0", hasError: false},
		{name: "zero_repeat_after_escape", input: `a\02`, expected: "a00", hasError: false},
		{name: "multiple_escapes_chain", input: `\\\3`, expected: `\3`, hasError: false},
		{name: "escape_before_digit", input: `a\2b3`, expected: "a2bbb", hasError: false},
		{name: "all_escaped_digits", input: `\1\2\3\4`, expected: "1234", hasError: false},
		{name: "mixed_unicode_escape", input: `😀\2a`, expected: "😀2a", hasError: false},
		{name: "unicode_emoji_repeat", input: `😀3`, expected: "😀😀😀", hasError: false},
		{name: "escape_unicode", input: `\😀2`, expected: "", hasError: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Unpack(tc.input)

			if tc.hasError {
				// Ожидаем ошибку для некорректных строк
				require.Error(t, err)
				require.True(t, errors.Is(err, ErrInvalidString))
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestUnpackConsecutiveDigits(t *testing.T) {
	// Тесты на последовательные цифры (должны быть ошибками)
	invalidCases := []struct {
		name  string
		input string
	}{
		{name: "two_digits", input: "a12"},
		{name: "three_digits", input: "a123"},
		{name: "digits_in_middle", input: "ab12cd"},
		{name: "digits_at_end", input: "abc23"},
		{name: "multiple_groups", input: "a12b34c56"},
		{name: "starts_with_multi_digit", input: "12abc"},
		{name: "only_multi_digit", input: "123"},
		{name: "digit_after_digit_no_char", input: "1a2"},
	}

	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Unpack(tc.input)
			// проверяем, что есть ошибка
			require.Error(t, err)
			// проверяем, что это наша ошибка
			require.True(t, errors.Is(err, ErrInvalidString),
				"expected ErrInvalidString for multi-digit number, got %v", err)
			require.Empty(t, result)
		})
	}
}
