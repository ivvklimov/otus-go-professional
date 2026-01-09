package hw03frequencyanalysis

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Change to true if needed.
var taskWithAsteriskIsCompleted = false

var text = `Как видите, он  спускается  по  лестнице  вслед  за  своим
	другом   Кристофером   Робином,   головой   вниз,  пересчитывая
	ступеньки собственным затылком:  бум-бум-бум.  Другого  способа
	сходить  с  лестницы  он  пока  не  знает.  Иногда ему, правда,
		кажется, что можно бы найти какой-то другой способ, если бы  он
	только   мог   на  минутку  перестать  бумкать  и  как  следует
	сосредоточиться. Но увы - сосредоточиться-то ему и некогда.
		Как бы то ни было, вот он уже спустился  и  готов  с  вами
	познакомиться.
	- Винни-Пух. Очень приятно!
		Вас,  вероятно,  удивляет, почему его так странно зовут, а
	если вы знаете английский, то вы удивитесь еще больше.
		Это необыкновенное имя подарил ему Кристофер  Робин.  Надо
	вам  сказать,  что  когда-то Кристофер Робин был знаком с одним
	лебедем на пруду, которого он звал Пухом. Для лебедя  это  было
	очень   подходящее  имя,  потому  что  если  ты  зовешь  лебедя
	громко: "Пу-ух! Пу-ух!"- а он  не  откликается,  то  ты  всегда
	можешь  сделать вид, что ты просто понарошку стрелял; а если ты
	звал его тихо, то все подумают, что ты  просто  подул  себе  на
	нос.  Лебедь  потом  куда-то делся, а имя осталось, и Кристофер
	Робин решил отдать его своему медвежонку, чтобы оно не  пропало
	зря.
		А  Винни - так звали самую лучшую, самую добрую медведицу
	в  зоологическом  саду,  которую  очень-очень  любил  Кристофер
	Робин.  А  она  очень-очень  любила  его. Ее ли назвали Винни в
	честь Пуха, или Пуха назвали в ее честь - теперь уже никто  не
	знает,  даже папа Кристофера Робина. Когда-то он знал, а теперь
	забыл.
		Словом, теперь мишку зовут Винни-Пух, и вы знаете почему.
		Иногда Винни-Пух любит вечерком во что-нибудь поиграть,  а
	иногда,  особенно  когда  папа  дома,  он больше любит тихонько
	посидеть у огня и послушать какую-нибудь интересную сказку.
		В этот вечер...`

func TestTop10(t *testing.T) {
	t.Run("no words in empty string", func(t *testing.T) {
		require.Len(t, Top10(""), 0)
	})

	t.Run("positive test", func(t *testing.T) {
		if taskWithAsteriskIsCompleted {
			expected := []string{
				"а",         // 8
				"он",        // 8
				"и",         // 6
				"ты",        // 5
				"что",       // 5
				"в",         // 4
				"его",       // 4
				"если",      // 4
				"кристофер", // 4
				"не",        // 4
			}
			require.Equal(t, expected, Top10(text))
		} else {
			expected := []string{
				"он",        // 8
				"а",         // 6
				"и",         // 6
				"ты",        // 5
				"что",       // 5
				"-",         // 4
				"Кристофер", // 4
				"если",      // 4
				"не",        // 4
				"то",        // 4
			}
			require.Equal(t, expected, Top10(text))
		}
	})
}

func TestTop10Custom(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "single word",
			input:    "hello",
			expected: []string{"hello"},
		},
		{
			name:     "example from assignment",
			input:    "cat and dog, one dog,two cats and one man",
			expected: []string{"and", "one", "cat", "cats", "dog,", "dog,two", "man"},
		},
		{
			name:     "multiple same frequency",
			input:    "b b b a a a c c c d e f g h i j k l m n o p",
			expected: []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"},
		},
		{
			name:     "case sensitive",
			input:    "Dog dog DOG",
			expected: []string{"DOG", "Dog", "dog"},
		},
		{
			name:     "with punctuation",
			input:    "hello! hello? hello. 'hello' \"hello\"",
			expected: []string{"\"hello\"", "'hello'", "hello!", "hello.", "hello?"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Top10(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d words, got %d", len(tt.expected), len(result))
			}
			for i, word := range tt.expected {
				if i >= len(result) {
					break
				}
				if result[i] != word {
					t.Errorf("word[%d]: expected %q, got %q", i, word, result[i])
				}
			}
		})
	}
}

func TestTop10Clean(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "case insensitive",
			input:    "Dog dog DOG",
			expected: []string{"dog"},
		},
		{
			name:     "remove punctuation",
			input:    "hello! hello? hello. 'hello' \"hello\"",
			expected: []string{"hello"},
		},
		{
			name:     "dash in middle remains",
			input:    "какой-то какойто",
			expected: []string{"какой-то", "какойто"},
		},
		{
			name:     "only punctuation",
			input:    "- ! ? .",
			expected: []string{},
		},
		{
			name:     "multiple dashes",
			input:    "--hello-- ---world---",
			expected: []string{"hello", "world"},
		},
		{
			name:     "combined",
			input:    "Hello, hello! HELLO? 'hello' \"HELLO\" --hello--",
			expected: []string{"hello"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Top10Clean(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d words, got %d: %v", len(tt.expected), len(result), result)
			}
			for i, word := range tt.expected {
				if i >= len(result) {
					break
				}
				if result[i] != word {
					t.Errorf("word[%d]: expected %q, got %q", i, word, result[i])
				}
			}
		})
	}
}
