package hw09structvalidator

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// Представляет ошибку валидации одного поля.
type ValidationError struct {
	Field string
	Err   error
}

// Слайс ошибок валидации, реализует интерфейс error.
type ValidationErrors []ValidationError

func (v ValidationErrors) Error() string {
	if len(v) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("validation failed")
	for i, ve := range v {
		if i == 0 {
			sb.WriteString(": ")
		} else {
			sb.WriteString("; ")
		}
		sb.WriteString(fmt.Sprintf("%s: %v", ve.Field, ve.Err))
	}
	return sb.String()
}

// Программные ошибки (не ошибки валидации данных).
var (
	ErrInvalidTag    = errors.New("invalid validate tag")
	ErrInvalidRegexp = errors.New("invalid regular expression")
)

// Валидирует публичные поля структуры по тэгу validate.
func Validate(v interface{}) error {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct, got %T", v)
	}

	var allErrors ValidationErrors
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		if !field.IsExported() {
			continue
		}

		tag := field.Tag.Get("validate")
		if tag == "" {
			continue
		}

		if tag == "nested" {
			if err := validateNestedField(field.Name, fieldVal, &allErrors); err != nil {
				return err
			}
			continue
		}

		validateSimpleField(field.Name, fieldVal, tag, &allErrors)
	}

	if len(allErrors) > 0 {
		return allErrors
	}
	return nil
}

func validateNestedField(fieldName string, fieldVal reflect.Value, allErrors *ValidationErrors) error {
	if fieldVal.Kind() != reflect.Struct {
		return nil
	}
	if err := Validate(fieldVal.Interface()); err != nil {
		var nestedErrs ValidationErrors
		if errors.As(err, &nestedErrs) {
			for _, e := range nestedErrs {
				*allErrors = append(*allErrors, ValidationError{
					Field: fieldName + "." + e.Field,
					Err:   e.Err,
				})
			}
			return nil
		}
		return err
	}
	return nil
}

func validateSimpleField(fieldName string, val reflect.Value, tag string, allErrors *ValidationErrors) {
	rules := strings.Split(tag, "|")

	//nolint:exhaustive
	switch val.Kind() {
	case reflect.String:
		s := val.String()
		for _, rule := range rules {
			if err := validateString(s, rule); err != nil {
				*allErrors = append(*allErrors, ValidationError{Field: fieldName, Err: err})
			}
		}

	case reflect.Int:
		n := val.Int()
		for _, rule := range rules {
			if err := validateInt(n, rule); err != nil {
				*allErrors = append(*allErrors, ValidationError{Field: fieldName, Err: err})
			}
		}

	case reflect.Slice:
		validateSliceField(fieldName, val, rules, allErrors)
	}
}

func validateSliceField(fieldName string, val reflect.Value, rules []string, allErrors *ValidationErrors) {
	elemKind := val.Type().Elem().Kind()
	if elemKind != reflect.String && elemKind != reflect.Int {
		return
	}

	for i := 0; i < val.Len(); i++ {
		elem := val.Index(i)
		elemName := fmt.Sprintf("%s[%d]", fieldName, i)

		if elemKind == reflect.String {
			for _, rule := range rules {
				if err := validateString(elem.String(), rule); err != nil {
					*allErrors = append(*allErrors, ValidationError{Field: elemName, Err: err})
				}
			}
		} else {
			for _, rule := range rules {
				if err := validateInt(elem.Int(), rule); err != nil {
					*allErrors = append(*allErrors, ValidationError{Field: elemName, Err: err})
				}
			}
		}
	}
}

func validateString(s, rule string) error {
	parts := strings.SplitN(rule, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("%w: %s", ErrInvalidTag, rule)
	}
	validator, arg := parts[0], parts[1]

	switch validator {
	case "len":
		n, err := strconv.Atoi(arg)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrInvalidTag, rule)
		}
		if len(s) != n {
			return fmt.Errorf("length must be %d, got %d", n, len(s))
		}

	case "regexp":
		re, err := regexp.Compile(arg)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrInvalidRegexp, arg)
		}
		if !re.MatchString(s) {
			return fmt.Errorf("does not match pattern %s", arg)
		}

	case "in":
		values := strings.Split(arg, ",")
		found := false
		for _, v := range values {
			if s == v {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("value must be one of [%s], got %s", arg, s)
		}

	default:
		return fmt.Errorf("%w: unknown validator %s", ErrInvalidTag, validator)
	}
	return nil
}

func validateInt(n int64, rule string) error {
	parts := strings.SplitN(rule, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("%w: %s", ErrInvalidTag, rule)
	}
	validator, arg := parts[0], parts[1]

	switch validator {
	case "min":
		minVal, err := strconv.ParseInt(arg, 10, 64)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrInvalidTag, rule)
		}
		if n < minVal {
			return fmt.Errorf("value must be >= %d, got %d", minVal, n)
		}

	case "max":
		maxVal, err := strconv.ParseInt(arg, 10, 64)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrInvalidTag, rule)
		}
		if n > maxVal {
			return fmt.Errorf("value must be <= %d, got %d", maxVal, n)
		}

	case "in":
		values := strings.Split(arg, ",")
		found := false
		for _, v := range values {
			val, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return fmt.Errorf("%w: %s", ErrInvalidTag, rule)
			}
			if n == val {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("value must be one of [%s], got %d", arg, n)
		}

	default:
		return fmt.Errorf("%w: unknown validator %s", ErrInvalidTag, validator)
	}
	return nil
}
