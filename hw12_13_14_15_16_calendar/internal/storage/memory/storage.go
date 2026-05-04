package memorystorage

import (
	"context"
	"sync"
	"time"

	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_calendar/internal/storage"
)

// Storage — реализация интерфейса storage.Storage в памяти.
// Использует map для хранения и sync.RWMutex для защиты от гонок данных.
type Storage struct {
	mu     sync.RWMutex
	events map[string]storage.Event
}

// New создаёт новое хранилище в памяти.
func New() *Storage {
	return &Storage{
		events: make(map[string]storage.Event),
	}
}

// CreateEvent добавляет событие.
// Перед добавлением проверяет, не пересекается ли время с другими событиями того же владельца.
func (s *Storage) CreateEvent(ctx context.Context, event *storage.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.events[event.ID]; exists {
		return storage.ErrDateBusy
	}

	if err := s.checkConflict(*event, ""); err != nil {
		return err
	}

	_ = ctx

	s.events[event.ID] = *event // Сохраняем копию значения
	return nil
}

// UpdateEvent изменяет событие.
func (s *Storage) UpdateEvent(ctx context.Context, id string, event storage.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Проверяем, существует ли событие
	if _, exists := s.events[id]; !exists {
		return storage.ErrNotFound
	}

	// Проверяем конфликты (исключая само себя по ID)
	event.ID = id // Убеждаемся, что ID обновляемого события совпадает
	if err := s.checkConflict(event, id); err != nil {
		return err
	}

	_ = ctx

	s.events[id] = event
	return nil
}

// DeleteEvent удаляет событие по ID.
func (s *Storage) DeleteEvent(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.events[id]; !exists {
		return storage.ErrNotFound
	}

	_ = ctx

	delete(s.events, id)
	return nil
}

// GetEvent получает событие по ID.
func (s *Storage) GetEvent(ctx context.Context, id string) (storage.Event, error) {
	s.mu.RLock() // Read Lock — можно читать параллельно
	defer s.mu.RUnlock()

	event, exists := s.events[id]
	if !exists {
		return storage.Event{}, storage.ErrNotFound
	}
	_ = ctx
	return event, nil
}

// ListEvents возвращает список событий владельца за период.
func (s *Storage) ListEvents(ctx context.Context, ownerID string, from, to time.Time) ([]storage.Event, error) {
	s.mu.RLock() // Read Lock
	defer s.mu.RUnlock()

	var result []storage.Event
	for _, event := range s.events {
		// Фильтр по владельцу
		if event.OwnerID != ownerID {
			continue
		}
		// Фильтр по времени (пересечение диапазонов)
		// Событие попадает в диапазон, если оно начинается раньше конца поиска
		// и заканчивается позже начала поиска
		if event.DateStart.Before(to) && event.DateEnd.After(from) {
			result = append(result, event)
		}
	}
	_ = ctx
	return result, nil
}

// checkConflict — внутренняя проверка пересечения времени.
// excludeID позволяет игнорировать событие с определённым ID (нужно при Update).
func (s *Storage) checkConflict(newEvent storage.Event, excludeID string) error {
	for _, existing := range s.events {
		// Пропускаем само себя при обновлении
		if existing.ID == excludeID {
			continue
		}

		// Проверяем только события того же владельца
		if existing.OwnerID != newEvent.OwnerID {
			continue
		}

		// Проверка пересечения интервалов:
		// (StartA < EndB) AND (EndA > StartB)
		if newEvent.DateStart.Before(existing.DateEnd) &&
			newEvent.DateEnd.After(existing.DateStart) {
			return storage.ErrDateBusy
		}
	}
	return nil
}
