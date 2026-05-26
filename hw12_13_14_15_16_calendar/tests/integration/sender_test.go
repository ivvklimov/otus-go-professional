//go:build integration

package integration

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateValidUUID генерирует валидный UUID v4 используя стандартную библиотеку.
// Формат: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx (где 4 = версия, y = 8/9/a/b)
func generateValidUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)

	// Set version (4) and variant (2 bits)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant RFC 4122

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// TestSender_MarkNotificationDelivered_Integration — полная проверка пайплайна уведомлений.
func TestSender_MarkNotificationDelivered_Integration(t *testing.T) {
	uniqueSuffix := time.Now().UnixNano() % 100000
	eventTitle := fmt.Sprintf("Notification Test-%d", uniqueSuffix)
	ownerID := int64(88888 + uniqueSuffix%1000)

	// Событие в будущем, уведомление нужно отправить СЕЙЧАС (в прошлом)
	eventStart := time.Now().Add(1 * time.Hour).Format(time.RFC3339)
	eventEnd := time.Now().Add(2 * time.Hour).Format(time.RFC3339)
	notifyAt := time.Now().Add(-5 * time.Minute).Format(time.RFC3339)

	eventID := createEventWithNotification(t, eventTitle, eventStart, eventEnd, ownerID, notifyAt)
	t.Logf("Created event with notification: id=%s, notify_at=%s", eventID, notifyAt)

	db, err := sql.Open("postgres", "postgres://intranet:password@localhost:5540/calendar_test?sslmode=disable")
	require.NoError(t, err, "Failed to connect to test database")
	defer db.Close()

	// Sanity check: убеждаемся, что запись создалась
	var storedNotifyAt string
	err = db.QueryRow("SELECT notify_at FROM events WHERE id = $1", eventID).Scan(&storedNotifyAt)
	require.NoError(t, err, "Event not found in DB after insertion")

	t.Log("Waiting for background workers (Scheduler/Sender) to process notification...")
	deadline := time.Now().Add(40 * time.Second)
	checkInterval := 2 * time.Second

	var deliveredAt sql.NullTime
	var publishedAt sql.NullTime

	for time.Now().Before(deadline) {
		err := db.QueryRow(
			"SELECT notify_published_at, notify_delivered_at FROM events WHERE id = $1",
			eventID,
		).Scan(&publishedAt, &deliveredAt)

		if err != nil && err != sql.ErrNoRows {
			time.Sleep(checkInterval)
			continue
		}

		if deliveredAt.Valid {
			t.Logf("SUCCESS: Sender marked notification as delivered at %s", deliveredAt.Time)
			break
		}

		if publishedAt.Valid {
			t.Logf("Scheduler worked (published_at=%s). Waiting for Sender to deliver...", publishedAt.Time)
		} else {
			t.Logf("Waiting for Scheduler to pick up event...")
		}
		time.Sleep(checkInterval)
	}

	// Если не прошло — выводим отладочную информацию
	if !deliveredAt.Valid {
		var dumpNotifyAt, dumpPublishedAt, dumpDeliveredAt sql.NullString
		_ = db.QueryRow("SELECT notify_at, notify_published_at, notify_delivered_at FROM events WHERE id = $1", eventID).
			Scan(&dumpNotifyAt, &dumpPublishedAt, &dumpDeliveredAt)

		t.Logf("FINAL DB STATE for event %s:", eventID)
		t.Logf("   notify_at:           %s", dumpNotifyAt.String)
		t.Logf("   notify_published_at: %s", dumpPublishedAt.String)
		t.Logf("   notify_delivered_at: %s (EXPECTED: NOT NULL)", dumpDeliveredAt.String)
		t.Log("Check sender logs: docker-compose -f deployments/docker/calendar/docker-compose.integration.yml logs otus_calendar_test")
	}

	assert.True(t, deliveredAt.Valid,
		"notify_delivered_at should be set by sender. Event ID: %s", eventID)
}

func createEventWithNotification(t *testing.T, title, dateStart, dateEnd string, ownerID int64, notifyAt string) string {
	t.Helper()
	db, err := sql.Open("postgres", "postgres://intranet:password@localhost:5540/calendar_test?sslmode=disable")
	require.NoError(t, err)
	defer db.Close()

	eventID := generateValidUUID()

	// Явно указываем NULL для полей уведомлений
	_, err = db.Exec(`
		INSERT INTO events (
			id, title, description, owner_id, 
			date_start, date_end, notify_at,
			notify_published_at, notify_delivered_at,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, NULL, NULL, NOW(), NOW())
	`, eventID, title, "Integration test notification", ownerID, dateStart, dateEnd, notifyAt)

	require.NoError(t, err, "Failed to insert test event. ID: %s", eventID)
	return eventID
}
