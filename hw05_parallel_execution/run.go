package hw05parallelexecution

import (
	"errors"
	"sync"
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

	errorCount := 0
	taskChanClosed := false // Флаг, что taskChan уже закрыт.

	startWorkers(&wg, n, len(tasks), taskChan, errChan, done)
	startController(&wg, tasks, n, m, taskChan, errChan, done, &errorCount, &taskChanClosed)

	wg.Wait()

	// Контроллер закрывает done и выходит при достижении лимита ошибок,
	// но некоторые воркеры могут ещё не завершиться. Когда wg.Wait() разблокируется,
	// мы проверяем накопившийся errorCount, чтобы вернуть правильный результат.
	// Без этой проверки функция могла бы вернуть nil даже при превышении лимита.
	if errorCount >= m {
		return ErrErrorsLimitExceeded
	}

	return nil
}

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

					err := task()
					select {
					case errChan <- err:
					case <-done:
					}
				}
			}
		}()
	}
}

func startController(
	wg *sync.WaitGroup,
	tasks []Task,
	n, m int,
	taskChan chan Task,
	errChan chan error,
	done chan struct{},
	errorCount *int,
	taskChanClosed *bool,
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
					*errorCount++
					if *errorCount >= m {
						close(done)
						return
					}
				}

				// Отправляем следующую задачу если есть.
				if sent < totalTasks {
					taskChan <- tasks[sent]
					sent++
				} else if sent == totalTasks && !*taskChanClosed {
					*taskChanClosed = true
					close(taskChan)
				}
			}
		}
	}()
}
