//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type CreateEventRequest struct {
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	DateStart   string  `json:"dateStart"`
	DateEnd     string  `json:"dateEnd"`
	OwnerID     int64   `json:"ownerId"`
	NotifyAt    *string `json:"notifyAt,omitempty"`
}

type EventResponse struct {
	Event struct {
		ID          string  `json:"id"`
		Title       string  `json:"title"`
		Description string  `json:"description"`
		OwnerID     int64   `json:"owner_id"`
		DateStart   string  `json:"date_start"`
		DateEnd     string  `json:"date_end"`
		NotifyAt    *string `json:"notify_at"`
		CreatedAt   string  `json:"created_at"`
		UpdatedAt   string  `json:"updated_at"`
	} `json:"event"`
}

type ListEventsResponse struct {
	Events []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	} `json:"events"`
}

// uniqueTimeSlot возвращает уникальные даты для теста, чтобы избежать конфликтов.
func uniqueTimeSlot(t *testing.T, dayOffset, hourOffset int) (start, end string) {
	t.Helper()
	base := time.Now().
		AddDate(0, 0, dayOffset).
		Add(time.Duration(hourOffset) * time.Hour).
		Truncate(time.Hour)
	return base.Format(time.RFC3339), base.Add(1 * time.Hour).Format(time.RFC3339)
}

func TestCreateEvent_Integration(t *testing.T) {
	url := testAPIURL + "/api/v1/calendar/events"
	start, end := uniqueTimeSlot(t, 10, 0)
	uniqueSuffix := time.Now().UnixNano() % 100000

	reqBody := CreateEventRequest{
		Title:     fmt.Sprintf("Integration Test Event-%d", uniqueSuffix),
		DateStart: start,
		DateEnd:   end,
		OwnerID:   12345 + uniqueSuffix%1000,
	}

	resp, err := httpPostJSON(url, reqBody)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		t.Logf("Response body: %s", string(body))
	}
	resp.Body = io.NopCloser(bytes.NewBuffer(body))

	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Expected 201, got %d. Body: %s", resp.StatusCode, string(body))

	var eventResp EventResponse
	err = json.Unmarshal(body, &eventResp)
	require.NoError(t, err, "Failed to parse response: %s", string(body))

	assert.NotEmpty(t, eventResp.Event.ID)
	t.Logf("Created event: id=%s, title=%s, owner=%d",
		eventResp.Event.ID, eventResp.Event.Title, eventResp.Event.OwnerID)
	assert.Contains(t, eventResp.Event.Title, "Integration Test Event")
}

func TestCreateEvent_Conflict_Integration(t *testing.T) {
	url := testAPIURL + "/api/v1/calendar/events"
	start, end := uniqueTimeSlot(t, 20, 0)
	uniqueSuffix := time.Now().UnixNano() % 100000

	// Создаём первое событие (база для конфликта)
	_ = createTestEvent(t, fmt.Sprintf("Conflict Test 1-%d", uniqueSuffix), start, end, 99999)

	// Формируем пересекающийся интервал
	overlapStart := time.Now().AddDate(0, 0, 20).Add(30 * time.Minute).Format(time.RFC3339)
	overlapEnd := time.Now().AddDate(0, 0, 20).Add(90 * time.Minute).Format(time.RFC3339)

	reqBody := CreateEventRequest{
		Title:     fmt.Sprintf("Conflict Test 2-%d", uniqueSuffix),
		DateStart: overlapStart,
		DateEnd:   overlapEnd,
		OwnerID:   99999,
	}

	resp, err := httpPostJSON(url, reqBody)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Contains(t, []int{http.StatusConflict, http.StatusBadRequest}, resp.StatusCode)

	t.Logf("Conflict correctly rejected: status=%d | Reason: time ranges overlap", resp.StatusCode)
	t.Logf("   Existing event: [%s, %s]", start, end)
	t.Logf("   New event:      [%s, %s] ← intersects with existing", overlapStart, overlapEnd)
}

func TestListEventsForDay_Integration(t *testing.T) {
	// Фиксируем один день для всех событий
	testDay := time.Now().AddDate(0, 0, 30).Truncate(24 * time.Hour)
	uniqueSuffix := time.Now().UnixNano() % 100000
	ownerID := int64(77777 + uniqueSuffix%1000)

	// Создаём 3 события в пределах одного дня
	_ = createTestEvent(t,
		fmt.Sprintf("Event 1-%d", uniqueSuffix),
		testDay.Add(9*time.Hour).Format(time.RFC3339),
		testDay.Add(10*time.Hour).Format(time.RFC3339),
		ownerID)
	_ = createTestEvent(t,
		fmt.Sprintf("Event 2-%d", uniqueSuffix),
		testDay.Add(11*time.Hour).Format(time.RFC3339),
		testDay.Add(12*time.Hour).Format(time.RFC3339),
		ownerID)
	_ = createTestEvent(t,
		fmt.Sprintf("Event 3-%d", uniqueSuffix),
		testDay.Add(14*time.Hour).Format(time.RFC3339),
		testDay.Add(15*time.Hour).Format(time.RFC3339),
		ownerID)

	dateParam := testDay.Format(time.RFC3339) // 2026-07-20T00:00:00Z

	url := testAPIURL + fmt.Sprintf("/api/v1/calendar/events/day?owner_id=%d&date=%s", ownerID, dateParam)
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Logf("ListEvents response: status=%d, body=%s", resp.StatusCode, string(body))
	}
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var listResp ListEventsResponse
	err = json.Unmarshal(body, &listResp)
	require.NoError(t, err)

	t.Logf("Found %d events for day %s", len(listResp.Events), dateParam)
	for _, e := range listResp.Events {
		t.Logf("  - %s (id=%s)", e.Title, e.ID)
	}

	// Проверяем, что наши события в списке
	titles := make(map[string]bool)
	for _, e := range listResp.Events {
		titles[e.Title] = true
	}

	expectedTitles := []string{
		fmt.Sprintf("Event 1-%d", uniqueSuffix),
		fmt.Sprintf("Event 2-%d", uniqueSuffix),
		fmt.Sprintf("Event 3-%d", uniqueSuffix),
	}
	for _, expected := range expectedTitles {
		assert.True(t, titles[expected], "Expected event '%s' not found in response", expected)
	}
}

// TestListEventsForWeek_Integration - листинг событий за неделю.
func TestListEventsForWeek_Integration(t *testing.T) {
	referenceDate := time.Now().AddDate(0, 0, 40).Truncate(24 * time.Hour)
	monday := referenceDate.Add(-time.Duration(referenceDate.Weekday()-1) * 24 * time.Hour)
	uniqueSuffix := time.Now().UnixNano() % 100000
	ownerID := int64(66666 + uniqueSuffix%1000)

	// 3 события внутри недели
	_ = createTestEvent(t, fmt.Sprintf("Week 1-%d", uniqueSuffix), monday.Add(24*time.Hour).Format(time.RFC3339), monday.Add(25*time.Hour).Format(time.RFC3339), ownerID)
	_ = createTestEvent(t, fmt.Sprintf("Week 2-%d", uniqueSuffix), monday.Add(48*time.Hour).Format(time.RFC3339), monday.Add(49*time.Hour).Format(time.RFC3339), ownerID)
	_ = createTestEvent(t, fmt.Sprintf("Week 3-%d", uniqueSuffix), monday.Add(72*time.Hour).Format(time.RFC3339), monday.Add(73*time.Hour).Format(time.RFC3339), ownerID)
	// 1 событие за пределами
	_ = createTestEvent(t, fmt.Sprintf("OutsideWeek-%d", uniqueSuffix), monday.Add(10*24*time.Hour).Format(time.RFC3339), monday.Add(11*24*time.Hour).Format(time.RFC3339), ownerID)

	dateParam := monday.Format(time.RFC3339)
	url := testAPIURL + fmt.Sprintf("/api/v1/calendar/events/week?owner_id=%d&date=%s", ownerID, dateParam)
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Week API error: %s", string(body))

	var listResp ListEventsResponse
	_ = json.Unmarshal(body, &listResp)

	t.Logf("Found %d events for week starting %s", len(listResp.Events), dateParam)
	for _, e := range listResp.Events {
		t.Logf("  - %s (id=%s)", e.Title, e.ID)
	}

	assert.GreaterOrEqual(t, len(listResp.Events), 3, "Expected at least 3 events inside the week")
}

// TestListEventsForMonth_Integration - листинг событий за месяц.
func TestListEventsForMonth_Integration(t *testing.T) {
	referenceDate := time.Now().AddDate(0, 2, 0)
	firstOfMonth := time.Date(referenceDate.Year(), referenceDate.Month(), 1, 0, 0, 0, 0, time.UTC)
	uniqueSuffix := time.Now().UnixNano() % 100000
	ownerID := int64(55555 + uniqueSuffix%1000)

	// 3 события внутри месяца
	_ = createTestEvent(t, fmt.Sprintf("Month 1-%d", uniqueSuffix), firstOfMonth.Add(5*24*time.Hour).Format(time.RFC3339), firstOfMonth.Add(6*24*time.Hour).Format(time.RFC3339), ownerID)
	_ = createTestEvent(t, fmt.Sprintf("Month 2-%d", uniqueSuffix), firstOfMonth.Add(15*24*time.Hour).Format(time.RFC3339), firstOfMonth.Add(16*24*time.Hour).Format(time.RFC3339), ownerID)
	_ = createTestEvent(t, fmt.Sprintf("Month 3-%d", uniqueSuffix), firstOfMonth.Add(25*24*time.Hour).Format(time.RFC3339), firstOfMonth.Add(26*24*time.Hour).Format(time.RFC3339), ownerID)
	// 1 событие за пределами
	_ = createTestEvent(t, fmt.Sprintf("OutsideMonth-%d", uniqueSuffix), firstOfMonth.AddDate(0, 1, 0).Format(time.RFC3339), firstOfMonth.AddDate(0, 1, 1).Format(time.RFC3339), ownerID)

	dateParam := firstOfMonth.Format(time.RFC3339)
	url := testAPIURL + fmt.Sprintf("/api/v1/calendar/events/month?owner_id=%d&date=%s", ownerID, dateParam)
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Month API error: %s", string(body))

	var listResp ListEventsResponse
	_ = json.Unmarshal(body, &listResp)

	t.Logf("Found %d events for month starting %s", len(listResp.Events), dateParam)
	for _, e := range listResp.Events {
		t.Logf("  - %s (id=%s)", e.Title, e.ID)
	}

	assert.GreaterOrEqual(t, len(listResp.Events), 3, "Expected at least 3 events inside the month")
}

func createTestEvent(t *testing.T, title, start, end string, ownerID int64) string {
	t.Helper()
	url := testAPIURL + "/api/v1/calendar/events"
	reqBody := CreateEventRequest{
		Title:     title,
		DateStart: start,
		DateEnd:   end,
		OwnerID:   ownerID,
	}

	resp, err := httpPostJSON(url, reqBody)
	require.NoError(t, err)
	defer resp.Body.Close()

	var eventResp EventResponse
	body, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(body, &eventResp)
	return eventResp.Event.ID
}
