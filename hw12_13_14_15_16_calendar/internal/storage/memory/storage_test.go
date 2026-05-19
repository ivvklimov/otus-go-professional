package memorystorage

import (
	"context"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/storage"
)

// ==================== Helpers ====================

func randomInt64() int64 {
	return rand.Int63n(1_000_000_000) + 1_000_000_000
}

// makeEvent создаёт тестовое событие с минимальным набором полей.
func makeEvent(id string, ownerID int64, start, end time.Time) storage.Event {
	return storage.Event{
		ID:        id,
		OwnerID:   ownerID,
		DateStart: start,
		DateEnd:   end,
		Title:     "Test Event",
	}
}

// stringPtr - хелпер для получения *string из string.
func stringPtr(s string) *string {
	return &s
}

// timeEqualMicro сравнивает два time.Time с точностью до микросекунд
// (для совместимости с тестами SQL-хранилища).
func timeEqualMicro(t1, t2 time.Time) bool {
	return t1.Truncate(time.Microsecond).Equal(t2.Truncate(time.Microsecond))
}

// ==================== ТЕСТЫ ====================

// TestMemoryStorage_CRUD проверяет базовые операции.
func TestMemoryStorage_CRUD(t *testing.T) {
	ctx := context.Background()
	store := New()

	owner := randomInt64()
	now := time.Now().UTC()

	// 1. CREATE
	event := makeEvent("1", owner, now, now.Add(time.Hour))
	err := store.CreateEvent(ctx, &event)
	require.NoError(t, err)

	// 2. GET
	gotEvent, err := store.GetEvent(ctx, "1")
	require.NoError(t, err)
	require.Equal(t, event.ID, gotEvent.ID)
	require.Equal(t, owner, gotEvent.OwnerID)

	// 3. UPDATE
	event.Title = "Updated Title"
	err = store.UpdateEvent(ctx, "1", event)
	require.NoError(t, err)

	gotEvent, _ = store.GetEvent(ctx, "1")
	require.Equal(t, "Updated Title", gotEvent.Title)

	// 4. DELETE
	err = store.DeleteEvent(ctx, "1")
	require.NoError(t, err)
	_, err = store.GetEvent(ctx, "1")
	require.ErrorIs(t, err, storage.ErrNotFound)
}

// TestMemoryStorage_DateConflict проверяет блокировку пересекающихся дат.
func TestMemoryStorage_DateConflict(t *testing.T) {
	ctx := context.Background()
	store := New()

	now := time.Now().UTC()
	owner := randomInt64()
	owner2 := randomInt64()

	// Создаем событие с 10:00 до 11:00
	ev1 := makeEvent("1", owner, now, now.Add(time.Hour))
	err := store.CreateEvent(ctx, &ev1)
	require.NoError(t, err)

	// Пытаемся создать событие с 10:30 до 11:30 (пересечение)
	ev2 := makeEvent("2", owner, now.Add(30*time.Minute), now.Add(90*time.Minute))
	err = store.CreateEvent(ctx, &ev2)
	require.ErrorIs(t, err, storage.ErrDateBusy, "expected conflict for overlapping dates")

	// Создаём НЕпересекающееся (12:00–13:00) - должно пройти
	ev3 := makeEvent("3", owner, now.Add(2*time.Hour), now.Add(3*time.Hour))
	err = store.CreateEvent(ctx, &ev3)
	require.NoError(t, err, "non-overlapping event should be created")

	// Тот же временной слот, но ДРУГОЙ владелец - должно пройти
	ev4 := makeEvent("4", owner2, now.Add(30*time.Minute), now.Add(90*time.Minute))
	err = store.CreateEvent(ctx, &ev4)
	require.NoError(t, err, "different owner should not conflict")
}

// TestMemoryStorage_ListEvents проверяет фильтрацию по владельцу и периоду.
func TestMemoryStorage_ListEvents(t *testing.T) {
	ctx := context.Background()
	store := New()

	ownerID := randomInt64()
	baseTime := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	// Создаём событие, которое ПОПАДАЕТ в диапазон (2 мая, 00:00–02:00)
	evtIn := makeEvent("in", ownerID, baseTime.Add(24*time.Hour), baseTime.Add(26*time.Hour))
	_ = store.CreateEvent(ctx, &evtIn)

	// Событие 1: 30 апреля (до диапазона)
	evBefore := makeEvent("before", ownerID, baseTime.Add(-2*time.Hour), baseTime.Add(-1*time.Hour))
	_ = store.CreateEvent(ctx, &evBefore)

	// Событие 2: 4 мая (после диапазона)
	evAfter := makeEvent("after", ownerID, baseTime.Add(72*time.Hour), baseTime.Add(74*time.Hour))
	_ = store.CreateEvent(ctx, &evAfter)

	// Запрашиваем события за 1–3 мая
	from := baseTime
	to := baseTime.Add(48 * time.Hour)

	events, err := store.ListEvents(ctx, ownerID, from, to)
	require.NoError(t, err)
	require.Len(t, events, 1, "expected exactly one event in range")
	require.Equal(t, evtIn.Title, events[0].Title)
	require.True(t, timeEqualMicro(evtIn.DateStart, events[0].DateStart))
	require.True(t, timeEqualMicro(evtIn.DateEnd, events[0].DateEnd))
}

// TestMemoryStorage_NotFound проверяет обработку несуществующих событий.
func TestMemoryStorage_NotFound(t *testing.T) {
	ctx := context.Background()
	store := New()

	nonExistentID := "00000000-0000-0000-0000-000000000000"

	_, err := store.GetEvent(ctx, nonExistentID)
	require.ErrorIs(t, err, storage.ErrNotFound)

	err = store.DeleteEvent(ctx, nonExistentID)
	require.ErrorIs(t, err, storage.ErrNotFound)

	err = store.UpdateEvent(ctx, nonExistentID, storage.Event{OwnerID: 12345})
	require.ErrorIs(t, err, storage.ErrNotFound)
}

// TestMemoryStorage_UpdatePartial_Title проверяет, что при обновлении только заголовка
// остальные поля (даты, description) сохраняются.
func TestMemoryStorage_UpdatePartial_Title(t *testing.T) {
	ctx := context.Background()
	store := New()

	ownerID := randomInt64()
	now := time.Now().UTC()

	// Создаём событие с полным набором полей
	evt := makeEvent("partial-title", ownerID, now, now.Add(time.Hour))
	evt.Description = stringPtr("original desc")
	evt.NotifyAt = &now
	err := store.CreateEvent(ctx, &evt)
	require.NoError(t, err)

	// Обновляем ТОЛЬКО заголовок (остальные поля - нулевые)
	updated := storage.Event{Title: "New Title"}
	err = store.UpdateEvent(ctx, "partial-title", updated)
	require.NoError(t, err)

	// Проверяем, что заголовок изменился, а остальное - нет
	got, err := store.GetEvent(ctx, "partial-title")
	require.NoError(t, err)
	require.Equal(t, "New Title", got.Title)
	require.Equal(t, "original desc", *got.Description)
	require.True(t, timeEqualMicro(evt.DateStart, got.DateStart))
	require.True(t, timeEqualMicro(evt.DateEnd, got.DateEnd))
}

// TestMemoryStorage_UpdatePartial_Dates проверяет обновление только временного диапазона.
func TestMemoryStorage_UpdatePartial_Dates(t *testing.T) {
	ctx := context.Background()
	store := New()

	ownerID := randomInt64()
	now := time.Now().UTC()

	evt := makeEvent("partial-dates", ownerID, now, now.Add(time.Hour))
	evt.Title = "Original Title"
	err := store.CreateEvent(ctx, &evt)
	require.NoError(t, err)

	// Обновляем ТОЛЬКО даты
	newStart := now.Add(2 * time.Hour)
	newEnd := now.Add(3 * time.Hour)
	updated := storage.Event{DateStart: newStart, DateEnd: newEnd}
	err = store.UpdateEvent(ctx, "partial-dates", updated)
	require.NoError(t, err)

	got, err := store.GetEvent(ctx, "partial-dates")
	require.NoError(t, err)
	require.Equal(t, "Original Title", got.Title) // заголовок не изменился
	require.True(t, timeEqualMicro(newStart, got.DateStart))
	require.True(t, timeEqualMicro(newEnd, got.DateEnd))
}

// TestMemoryStorage_UpdateConflict проверяет, что обновление, приводящее к пересечению,
// возвращает ErrDateBusy.
func TestMemoryStorage_UpdateConflict(t *testing.T) {
	ctx := context.Background()
	store := New()

	ownerID := randomInt64()
	now := time.Now().UTC()

	// Создаём два непересекающихся события
	evt1 := makeEvent("conflict-1", ownerID, now, now.Add(time.Hour)) // 10:00–11:00
	err := store.CreateEvent(ctx, &evt1)
	require.NoError(t, err)

	evt2 := makeEvent("conflict-2", ownerID, now.Add(2*time.Hour), now.Add(3*time.Hour)) // 12:00–13:00
	err = store.CreateEvent(ctx, &evt2)
	require.NoError(t, err)

	// Пытаемся сдвинуть evt2 так, чтобы он пересёкся с evt1 (10:30–11:30)
	conflictStart := now.Add(30 * time.Minute)
	conflictEnd := now.Add(90 * time.Minute)
	updated := storage.Event{DateStart: conflictStart, DateEnd: conflictEnd}
	err = store.UpdateEvent(ctx, "conflict-2", updated)
	require.ErrorIs(t, err, storage.ErrDateBusy, "expected conflict when updating to overlapping dates")
}

// TestMemoryStorage_ListEvents_Empty проверяет возврат пустого списка, если событий нет.
func TestMemoryStorage_ListEvents_Empty(t *testing.T) {
	ctx := context.Background()
	store := New()

	ownerID := randomInt64()
	from := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)

	events, err := store.ListEvents(ctx, ownerID, from, to)
	require.NoError(t, err)
	require.Empty(t, events, "expected empty list when no events exist")
}

// TestMemoryStorage_ListEvents_OwnerFilter проверяет, что события другого владельца не возвращаются.
func TestMemoryStorage_ListEvents_OwnerFilter(t *testing.T) {
	ctx := context.Background()
	store := New()

	owner1 := randomInt64()
	owner2 := randomInt64()
	now := time.Now().UTC()

	// Создаём события для двух владельцев в один и тот же временной слот
	evt1 := makeEvent("owner1", owner1, now, now.Add(time.Hour))
	_ = store.CreateEvent(ctx, &evt1)

	evt2 := makeEvent("owner2", owner2, now, now.Add(time.Hour))
	_ = store.CreateEvent(ctx, &evt2)

	// Запрашиваем только для owner1
	events, err := store.ListEvents(ctx, owner1, now, now.Add(2*time.Hour))
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, owner1, events[0].OwnerID)
}

// TestMemoryStorage_Concurrency проверяет потокобезопасность.
// Запускать с флагом -race!
func TestMemoryStorage_Concurrency(t *testing.T) {
	ctx := context.Background()
	store := New()
	var wg sync.WaitGroup

	owner := randomInt64()

	// Запускаем 100 горутин на запись
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			now := time.Now().Add(time.Duration(id) * time.Hour)
			ev := makeEvent(string(rune(id)), owner, now, now.Add(time.Minute))
			// Мы не проверяем ошибки здесь, так как конфликты возможны,
			// главное чтобы не было паники или гонки данных
			_ = store.CreateEvent(ctx, &ev)
		}(i)
	}

	_ = t
	wg.Wait()
}
