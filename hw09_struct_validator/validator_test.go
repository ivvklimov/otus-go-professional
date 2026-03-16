package hw09structvalidator

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

type UserRole string

// Test the function on different structures and other types.
type (
	User struct {
		ID     string `json:"id" validate:"len:36"`
		Name   string
		Age    int             `validate:"min:18|max:50"`
		Email  string          `validate:"regexp:^\\w+@\\w+\\.\\w+$"`
		Role   UserRole        `validate:"in:admin,stuff"`
		Phones []string        `validate:"len:11"`
		meta   json.RawMessage //nolint:unused
	}

	App struct {
		Version string `validate:"len:5"`
	}

	Token struct {
		Header    []byte
		Payload   []byte
		Signature []byte
	}

	Response struct {
		Code int    `validate:"in:200,404,500"`
		Body string `json:"omitempty"`
	}
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		in          interface{}
		expectedErr error
		wantFields  []string
	}{
		{
			name: "валидный User",
			in: User{
				ID:     "123456789012345678901234567890123456",
				Age:    25,
				Email:  "test@example.com",
				Role:   "admin",
				Phones: []string{"12345678901"},
			},
			expectedErr: nil,
		},
		{
			name: "User с несколькими ошибками валидации",
			in: User{
				ID:     "short",
				Age:    10,
				Email:  "invalid",
				Role:   "guest",
				Phones: []string{"123", "12345678901"},
			},
			expectedErr: ValidationErrors{},
			wantFields:  []string{"ID", "Age", "Email", "Role", "Phones[0]"},
		},
		{
			name:        "валидный App",
			in:          App{Version: "1.2.3"},
			expectedErr: nil,
		},
		{
			name:        "невалидная версия App",
			in:          App{Version: "1.2"},
			expectedErr: ValidationErrors{},
			wantFields:  []string{"Version"},
		},
		{
			name:        "структура без тэгов validate (игнорируется)",
			in:          Token{Header: []byte("test")},
			expectedErr: nil,
		},
		{
			name:        "валидный код ответа",
			in:          Response{Code: 200},
			expectedErr: nil,
		},
		{
			name:        "невалидный код ответа",
			in:          Response{Code: 400},
			expectedErr: ValidationErrors{},
			wantFields:  []string{"Code"},
		},
		{
			name:        "входное значение — не структура",
			in:          "not a struct",
			expectedErr: fmt.Errorf("expected struct"),
		},
		{
			name: "невалидная регулярка (программная ошибка)",
			in: struct {
				Field string `validate:"regexp:[invalid"`
			}{},
			expectedErr: ErrInvalidRegexp,
		},
		{
			name: "неверный формат тэга (программная ошибка)",
			in: struct {
				Field string `validate:"unknown"`
			}{},
			expectedErr: ErrInvalidTag,
		},
		{
			name: "комбинация правил: число вне диапазона",
			in: struct {
				Value int `validate:"min:0|max:10"`
			}{Value: 15},
			expectedErr: ValidationErrors{},
			wantFields:  []string{"Value"},
		},
		{
			name: "комбинация правил: строка не соответствует обоим условиям",
			in: struct {
				Value string `validate:"regexp:^\\d+$|len:5"`
			}{Value: "abcde"},
			expectedErr: ValidationErrors{},
			wantFields:  []string{"Value"},
		},
		{
			name: "слайс: один элемент не проходит валидацию",
			in: struct {
				Numbers []int `validate:"min:10"`
			}{Numbers: []int{5, 15, 20}},
			expectedErr: ValidationErrors{},
			wantFields:  []string{"Numbers[0]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runValidateTest(t, tt)
		})
	}
}

func runValidateTest(
	t *testing.T,
	tt struct {
		name        string
		in          interface{}
		expectedErr error
		wantFields  []string
	},
) {
	t.Helper()

	err := Validate(tt.in)

	if tt.expectedErr == nil {
		if err != nil {
			t.Errorf("Validate() = %v, want nil", err)
		}
		return
	}

	if err == nil {
		t.Errorf("Validate() = nil, want error")
		return
	}

	if strings.Contains(tt.name, "не структура") {
		if !strings.Contains(err.Error(), "expected struct") {
			t.Errorf("Validate() error = %v, want error containing 'expected struct'", err)
		}
		return
	}

	if errors.Is(err, ErrInvalidRegexp) || errors.Is(err, ErrInvalidTag) {
		if !errors.Is(err, tt.expectedErr) {
			t.Errorf("Validate() error = %v, want %v", err, tt.expectedErr)
		}
		return
	}

	validateErrors(t, err, tt.wantFields)
}

func validateErrors(t *testing.T, err error, wantFields []string) {
	t.Helper()

	var ve ValidationErrors
	if !errors.As(err, &ve) {
		t.Errorf("Validate() error type = %T, want ValidationErrors", err)
		return
	}

	for _, want := range wantFields {
		found := false
		for _, e := range ve {
			if e.Field == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing validation error for field %q, got errors: %v", want, ve)
		}
	}
}

func TestValidateNested(t *testing.T) {
	type Meta struct {
		Key  string   `validate:"len:10"`
		Code int      `validate:"min:100|max:999"`
		Tags []string `validate:"len:3"`
	}
	type UserWithMeta struct {
		Name string `validate:"len:5"`
		Meta Meta   `validate:"nested"`
	}
	type Profile struct {
		User   UserWithMeta `validate:"nested"`
		Active bool
	}

	tests := []struct {
		name       string
		in         interface{}
		wantFields []string
		wantErr    bool
	}{
		{
			name: "валидная вложенная структура",
			in: UserWithMeta{
				Name: "Johny",
				Meta: Meta{
					Key:  "1234567890",
					Code: 200,
					Tags: []string{"abc", "def", "ghi"},
				},
			},
			wantFields: nil,
			wantErr:    false,
		},
		{
			name: "ошибки в родительской и вложенной структуре",
			in: UserWithMeta{
				Name: "Jo",
				Meta: Meta{
					Key:  "short",
					Code: 50,
					Tags: []string{"a"},
				},
			},
			wantFields: []string{"Name", "Meta.Key", "Meta.Code", "Meta.Tags[0]"},
			wantErr:    true,
		},
		{
			name: "глубокая вложенность",
			in: Profile{
				User: UserWithMeta{
					Name: "Jo",
					Meta: Meta{
						Key:  "ok12345678",
						Code: 200,
						Tags: []string{"x", "y", "z"},
					},
				},
				Active: true,
			},
			wantFields: []string{"User.Name"},
			wantErr:    true,
		},
		{
			name: "вложенная структура с программной ошибкой",
			in: struct {
				Inner struct {
					Field string `validate:"regexp:[bad"`
				} `validate:"nested"`
			}{},
			wantFields: nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runNestedTest(t, tt)
		})
	}
}

func runNestedTest(
	t *testing.T,
	tt struct {
		name       string
		in         interface{}
		wantFields []string
		wantErr    bool
	},
) {
	t.Helper()

	err := Validate(tt.in)

	if !tt.wantErr {
		if err != nil {
			t.Errorf("Validate() = %v, want nil", err)
		}
		return
	}

	if err == nil {
		t.Errorf("Validate() = nil, want error")
		return
	}

	if errors.Is(err, ErrInvalidRegexp) || errors.Is(err, ErrInvalidTag) {
		return
	}

	validateErrors(t, err, tt.wantFields)
}
