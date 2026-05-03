package hw05parallelexecution

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestRun(t *testing.T) {
	defer goleak.VerifyNone(t)

	t.Run("if were errors in first M tasks, than finished not more N+M tasks", func(t *testing.T) {
		tasksCount := 50
		tasks := make([]Task, 0, tasksCount)

		var runTasksCount int32

		for i := 0; i < tasksCount; i++ {
			err := fmt.Errorf("error from task %d", i)
			tasks = append(tasks, func() error {
				time.Sleep(time.Millisecond * time.Duration(rand.Intn(100)))
				atomic.AddInt32(&runTasksCount, 1)
				return err
			})
		}

		workersCount := 10
		maxErrorsCount := 23
		err := Run(tasks, workersCount, maxErrorsCount)

		require.Truef(t, errors.Is(err, ErrErrorsLimitExceeded), "actual err - %v", err)
		require.LessOrEqual(t, runTasksCount, int32(workersCount+maxErrorsCount), "extra tasks were started")
	})

	t.Run("tasks without errors", func(t *testing.T) {
		tasksCount := 50
		tasks := make([]Task, 0, tasksCount)

		var runTasksCount int32
		var sumTime time.Duration

		for i := 0; i < tasksCount; i++ {
			taskSleep := time.Millisecond * time.Duration(rand.Intn(100))
			sumTime += taskSleep

			tasks = append(tasks, func() error {
				time.Sleep(taskSleep)
				atomic.AddInt32(&runTasksCount, 1)
				return nil
			})
		}

		workersCount := 5
		maxErrorsCount := 1

		start := time.Now()
		err := Run(tasks, workersCount, maxErrorsCount)
		elapsedTime := time.Since(start)
		require.NoError(t, err)

		require.Equal(t, int32(tasksCount), runTasksCount, "not all tasks were completed")
		require.LessOrEqual(t, int64(elapsedTime), int64(sumTime/2), "tasks were run sequentially?")
	})
}

// TestConcurrencyWithoutSleep проверяет, что задачи выполняются параллельно без использования time.Sleep.
func TestConcurrencyWithoutSleep(t *testing.T) {
	defer goleak.VerifyNone(t)

	t.Run("simple concurrency check", func(t *testing.T) {
		const workers = 3

		// Канал для подсчета запущенных задач.
		started := make(chan struct{}, workers)
		barrier := make(chan struct{})

		tasks := make([]Task, 0, workers)
		for i := 0; i < workers; i++ {
			tasks = append(tasks, func() error {
				started <- struct{}{} // Сигнал: задача запустилась.
				<-barrier             // Блокируем.
				return nil
			})
		}

		// Запускаем Run.
		go Run(tasks, workers, 1)

		// Ждем пока workers задач начнут выполнение.
		for i := 0; i < workers; i++ {
			select {
			case <-started:
				// OK.
			case <-time.After(time.Second):
				t.Fatalf("задачи не запускаются параллельно")
			}
		}

		// Если дошли сюда - значит workers задач запустились параллельно!
		close(barrier)
	})

	t.Run("early exit on errors doesn't leave hanging goroutines", func(t *testing.T) {
		const (
			workers    = 3
			maxErrors  = 2
			totalTasks = 10
		)

		var tasksStarted int32

		// Используем мьютекс и кондейшен для точного контроля
		var mu sync.Mutex
		cond := sync.NewCond(&mu)
		shouldRun := false
		runCount := 0

		tasks := make([]Task, 0, totalTasks)
		for i := 0; i < totalTasks; i++ {
			tasks = append(tasks, func() error {
				atomic.AddInt32(&tasksStarted, 1)

				mu.Lock()
				// Ждем разрешения на выполнение
				for !shouldRun {
					cond.Wait()
				}
				runCount++
				mu.Unlock()

				return errors.New("planned error")
			})
		}

		// Запускаем выполнение
		errChan := make(chan error, 1)
		go func() {
			errChan <- Run(tasks, workers, maxErrors)
		}()

		// Ждем, пока запустятся первые workers задач
		require.Eventually(t, func() bool {
			return atomic.LoadInt32(&tasksStarted) >= int32(workers)
		}, time.Second, time.Millisecond)

		// Разрешаем выполнение
		mu.Lock()
		shouldRun = true
		cond.Broadcast()
		mu.Unlock()

		// Ждем результат
		err := <-errChan
		require.Error(t, err)
		require.Equal(t, ErrErrorsLimitExceeded, err)

		// Проверяем счетчик выполненных задач
		mu.Lock()
		completed := runCount
		mu.Unlock()

		require.LessOrEqual(t, completed, workers+maxErrors,
			"не должно быть выполнено больше чем workers + maxErrors задач")
	})
}

// TestEdgeCases проверяет граничные случаи выполнения.
func TestEdgeCases(t *testing.T) {
	defer goleak.VerifyNone(t)

	t.Run("zero workers does nothing", func(t *testing.T) {
		tasks := []Task{
			func() error {
				t.Error("task should not be executed with zero workers")
				return nil
			},
		}
		err := Run(tasks, 0, 1)
		require.NoError(t, err)
	})

	t.Run("m=0 stops as soon as possible", func(t *testing.T) {
		var runCount int32

		tasks := []Task{
			func() error {
				atomic.AddInt32(&runCount, 1)
				return errors.New("error 1")
			},
			func() error {
				atomic.AddInt32(&runCount, 1)
				return errors.New("error 2")
			},
			func() error {
				atomic.AddInt32(&runCount, 1)
				return nil
			},
		}

		err := Run(tasks, 2, 0)
		require.Error(t, err)
		require.Equal(t, ErrErrorsLimitExceeded, err)

		// При m=0 может выполниться до n задач, пока система не обнаружит ошибку
		// Поэтому проверяем, что выполнилось не больше чем n задач
		require.LessOrEqual(t, atomic.LoadInt32(&runCount), int32(2),
			"should stop quickly when m=0, executed: %d", atomic.LoadInt32(&runCount))
	})

	t.Run("empty tasks list returns immediately", func(t *testing.T) {
		start := time.Now()
		err := Run([]Task{}, 5, 2)
		elapsed := time.Since(start)

		require.NoError(t, err)
		require.Less(t, elapsed, time.Millisecond*10,
			"should return immediately for empty tasks")
	})

	t.Run("tasks less than workers", func(t *testing.T) {
		const tasksCount = 2
		const workers = 5

		var runCount int32
		tasks := make([]Task, 0, tasksCount)

		for i := 0; i < tasksCount; i++ {
			tasks = append(tasks, func() error {
				atomic.AddInt32(&runCount, 1)
				return nil
			})
		}

		err := Run(tasks, workers, 1)
		require.NoError(t, err)
		require.Equal(t, int32(tasksCount), atomic.LoadInt32(&runCount),
			"all tasks should be executed even if workers > tasks")
	})

	t.Run("all tasks error, stop at m", func(t *testing.T) {
		const totalTasks = 10
		const workers = 3
		const maxErrors = 2

		var runCount int32
		tasks := make([]Task, 0, totalTasks)

		for i := 0; i < totalTasks; i++ {
			tasks = append(tasks, func() error {
				atomic.AddInt32(&runCount, 1)
				return errors.New("error")
			})
		}

		err := Run(tasks, workers, maxErrors)
		require.Error(t, err)
		require.Equal(t, ErrErrorsLimitExceeded, err)

		// Выполнится не более чем workers + maxErrors задач
		require.LessOrEqual(t, atomic.LoadInt32(&runCount),
			int32(workers+maxErrors),
			"should stop after m errors even if all tasks error")
	})
}
