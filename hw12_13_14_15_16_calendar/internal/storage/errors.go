package storage

import "errors"

var (
	// ErrDateBusy возвращается, если создаваемое событие пересекается по времени с уже существующим.
	ErrDateBusy = errors.New("date busy")

	// ErrNotFound возвращается, если событие с указанным ID не найдено.
	ErrNotFound = errors.New("event not found")

	// ErrInvalidID возвращается, если ID имеет неверный формат.
	ErrInvalidID = errors.New("invalid event id")
)
