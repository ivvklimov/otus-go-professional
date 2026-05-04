package memorystorage

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_calendar/internal/storage"
)

// Вспомогательная функция для создания событий.
func makeEvent(id, ownerID string, start, end time.Time) storage.Event {
	return storage.Event{
		ID:        id,
		OwnerID:   ownerID,
		DateStart: start,
		DateEnd:   end,
		Title:     "Test Event",
	}
}

func TestMemoryStorage_CRUD(t *testing.T) {
	ctx := context.Background()
	store := New()

	// 1. CREATE
	event := makeEvent("1", "owner_1", time.Now(), time.Now().Add(time.Hour))
	if err := store.CreateEvent(ctx, &event); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// 2. GET
	gotEvent, err := store.GetEvent(ctx, "1")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if gotEvent.ID != event.ID {
		t.Fatalf("expected ID %s, got %s", event.ID, gotEvent.ID)
	}

	// 3. UPDATE
	event.Title = "Updated Title"
	if err := store.UpdateEvent(ctx, "1", event); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	gotEvent, _ = store.GetEvent(ctx, "1")
	if gotEvent.Title != "Updated Title" {
		t.Fatalf("update didn't work")
	}

	// 4. DELETE
	if err := store.DeleteEvent(ctx, "1"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	_, err = store.GetEvent(ctx, "1")
	if err != storage.ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestMemoryStorage_DateConflict(t *testing.T) {
	ctx := context.Background()
	store := New()

	now := time.Now()
	owner := "owner_1"

	// Создаем событие с 10:00 до 11:00
	ev1 := makeEvent("1", owner, now, now.Add(time.Hour))
	if err := store.CreateEvent(ctx, &ev1); err != nil {
		t.Fatal(err)
	}

	// Пытаемся создать событие с 10:30 до 11:30 (пересечение)
	ev2 := makeEvent("2", owner, now.Add(30*time.Minute), now.Add(90*time.Minute))
	err := store.CreateEvent(ctx, &ev2)

	if err != storage.ErrDateBusy {
		t.Fatalf("expected ErrDateBusy, got %v", err)
	}

	// Пытаемся создать событие с 12:00 до 13:00 (НЕТ пересечения)
	ev3 := makeEvent("3", owner, now.Add(2*time.Hour), now.Add(3*time.Hour))
	if err := store.CreateEvent(ctx, &ev3); err != nil {
		t.Fatalf("should not be busy, got: %v", err)
	}

	// Пытаемся создать событие с тем же временем, но ДРУГОЙ владелец (должно пройти)
	ev4 := makeEvent("4", "owner_2", now.Add(30*time.Minute), now.Add(90*time.Minute))
	if err := store.CreateEvent(ctx, &ev4); err != nil {
		t.Fatalf("different owner should not conflict, got: %v", err)
	}
}

// TestMemoryStorage_Concurrency проверяет потокобезопасность
// Запускать с флагом -race!
func TestMemoryStorage_Concurrency(t *testing.T) {
	ctx := context.Background()
	store := New()
	var wg sync.WaitGroup

	// Запускаем 100 горутин на запись
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			now := time.Now().Add(time.Duration(id) * time.Hour)
			ev := makeEvent(string(rune(id)), "owner_conc", now, now.Add(time.Minute))
			// Мы не проверяем ошибки здесь, так как конфликты возможны,
			// главное чтобы не было паники или гонки данных
			_ = store.CreateEvent(ctx, &ev)
		}(i)
	}

	_ = t
	wg.Wait()
}
