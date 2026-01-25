package hw05parallelexecution

import (
	"errors"
	"sync"
	"sync/atomic"
)

var ErrErrorsLimitExceeded = errors.New("errors limit exceeded")

type Task func() error

func Run(tasks []Task, n, m int) error {
	if n <= 0 {
		return nil
	}

	if m <= 0 {
		m = 1
	}

	taskChan := make(chan Task)
	errChan := make(chan error, n)
	done := make(chan struct{})
	var wg sync.WaitGroup

	var errorCount int32
	var taskChanClosed int32 // Флаг, что taskChan уже закрыт.

	startWorkers(&wg, n, len(tasks), taskChan, errChan, done)
	startController(&wg, tasks, n, m, taskChan, errChan, done, &errorCount, &taskChanClosed)

	wg.Wait()

	// Контроллер закрывает done и выходит при достижении лимита ошибок,
	// но некоторые воркеры могут ещё не завершиться. Когда wg.Wait() разблокируется,
	// мы проверяем накопившийся errorCount, чтобы вернуть правильный результат.
	// Без этой проверки функция могла бы вернуть nil даже при превышении лимита.
	if int(atomic.LoadInt32(&errorCount)) >= m {
		return ErrErrorsLimitExceeded
	}

	return nil
}

//nolint:gocognit
func startWorkers(
	wg *sync.WaitGroup,
	n, taskLen int,
	taskChan chan Task,
	errChan chan error,
	done chan struct{},
) {
	// Воркеры.
	for i := 0; i < n && i < taskLen; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				case task, ok := <-taskChan:
					if !ok {
						return
					}

					if err := task(); err != nil {
						select {
						case errChan <- err:
						case <-done:
						}
					} else {
						select {
						case errChan <- nil:
						case <-done:
						}
					}
				}
			}
		}()
	}
}

//nolint:gocognit
func startController(
	wg *sync.WaitGroup,
	tasks []Task,
	n, m int,
	taskChan chan Task,
	errChan chan error,
	done chan struct{},
	errorCount, taskChanClosed *int32,
) {
	// Контроллер.
	wg.Add(1)
	go func() {
		defer wg.Done()

		totalTasks := len(tasks)
		sent := 0
		completed := 0

		// Отправляем первые n задач.
		for sent < n && sent < totalTasks {
			select {
			case <-done:
				return
			case taskChan <- tasks[sent]:
				sent++
			}
		}

		// Обрабатываем результаты.
		for completed < totalTasks {
			select {
			case <-done:
				return
			case err, ok := <-errChan:
				if !ok {
					// errChan закрыт - аварийный выход.
					return
				}

				completed++ // Увеличиваем только после проверки ok.

				if err != nil {
					newCount := atomic.AddInt32(errorCount, 1)
					if int(newCount) >= m {
						close(done)
						return
					}
				}

				// Отправляем следующую задачу если есть.
				if sent < totalTasks {
					select {
					case <-done:
						return
					case taskChan <- tasks[sent]:
						sent++
					}
				} else if sent == totalTasks && atomic.CompareAndSwapInt32(taskChanClosed, 0, 1) {
					// Закрываем taskChan только один раз.
					close(taskChan)
				}
			}
		}
	}()
}
