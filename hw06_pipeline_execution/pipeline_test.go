package hw06pipelineexecution

import (
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	sleepPerStage = time.Millisecond * 100
	fault         = sleepPerStage / 2
)

func TestPipeline(t *testing.T) {
	// Stage generator
	g := func(_ string, f func(v interface{}) interface{}) Stage {
		return func(in In) Out {
			out := make(Bi)
			go func() {
				defer close(out)
				for v := range in {
					time.Sleep(sleepPerStage)
					out <- f(v)
				}
			}()
			return out
		}
	}

	stages := []Stage{
		g("Dummy", func(v interface{}) interface{} { return v }),
		g("Multiplier (* 2)", func(v interface{}) interface{} { return v.(int) * 2 }),
		g("Adder (+ 100)", func(v interface{}) interface{} { return v.(int) + 100 }),
		g("Stringifier", func(v interface{}) interface{} { return strconv.Itoa(v.(int)) }),
	}

	t.Run("simple case", func(t *testing.T) {
		in := make(Bi)
		data := []int{1, 2, 3, 4, 5}

		go func() {
			for _, v := range data {
				in <- v
			}
			close(in)
		}()

		result := make([]string, 0, 10)
		start := time.Now()
		for s := range ExecutePipeline(in, nil, stages...) {
			result = append(result, s.(string))
		}
		elapsed := time.Since(start)

		require.Equal(t, []string{"102", "104", "106", "108", "110"}, result)
		require.Less(t,
			int64(elapsed),
			// ~0.8s for processing 5 values in 4 stages (100ms every) concurrently
			int64(sleepPerStage)*int64(len(stages)+len(data)-1)+int64(fault))
	})

	t.Run("done case", func(t *testing.T) {
		in := make(Bi)
		done := make(Bi)
		data := []int{1, 2, 3, 4, 5}

		// Abort after 200ms
		abortDur := sleepPerStage * 2
		go func() {
			<-time.After(abortDur)
			close(done)
		}()

		go func() {
			for _, v := range data {
				in <- v
			}
			close(in)
		}()

		result := make([]string, 0, 10)
		start := time.Now()
		for s := range ExecutePipeline(in, done, stages...) {
			result = append(result, s.(string))
		}
		elapsed := time.Since(start)

		require.Len(t, result, 0)
		require.Less(t, int64(elapsed), int64(abortDur)+int64(fault))
	})
}

func TestAllStageStop(t *testing.T) {
	wg := sync.WaitGroup{}
	// Stage generator
	g := func(_ string, f func(v interface{}) interface{}) Stage {
		return func(in In) Out {
			out := make(Bi)
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer close(out)
				for v := range in {
					time.Sleep(sleepPerStage)
					out <- f(v)
				}
			}()
			return out
		}
	}

	stages := []Stage{
		g("Dummy", func(v interface{}) interface{} { return v }),
		g("Multiplier (* 2)", func(v interface{}) interface{} { return v.(int) * 2 }),
		g("Adder (+ 100)", func(v interface{}) interface{} { return v.(int) + 100 }),
		g("Stringifier", func(v interface{}) interface{} { return strconv.Itoa(v.(int)) }),
	}

	t.Run("done case", func(t *testing.T) {
		in := make(Bi)
		done := make(Bi)
		data := []int{1, 2, 3, 4, 5}

		// Abort after 200ms
		abortDur := sleepPerStage * 2
		go func() {
			<-time.After(abortDur)
			close(done)
		}()

		go func() {
			for _, v := range data {
				in <- v
			}
			close(in)
		}()

		result := make([]string, 0, 10)
		for s := range ExecutePipeline(in, done, stages...) {
			result = append(result, s.(string))
		}
		wg.Wait()

		require.Len(t, result, 0)
	})
}

func TestPipelineEmptyStages(t *testing.T) {
	// Проверяем, что пайплайн без стадий просто пропускает данные
	t.Run("no stages", func(t *testing.T) {
		in := make(Bi)
		data := []int{1, 2, 3}

		go func() {
			for _, v := range data {
				in <- v
			}
			close(in)
		}()

		result := make([]int, 0, 3)
		for v := range ExecutePipeline(in, nil) {
			result = append(result, v.(int))
		}

		require.Equal(t, []int{1, 2, 3}, result)
	})

	// Проверяем корректную работу с одной стадией
	t.Run("single stage", func(t *testing.T) {
		in := make(Bi)
		data := []int{1, 2, 3}

		stage := func(in In) Out {
			out := make(Bi)
			go func() {
				defer close(out)
				for v := range in {
					out <- v.(int) * 2
				}
			}()
			return out
		}

		go func() {
			for _, v := range data {
				in <- v
			}
			close(in)
		}()

		result := make([]int, 0, 3)
		for v := range ExecutePipeline(in, nil, stage) {
			result = append(result, v.(int))
		}

		require.Equal(t, []int{2, 4, 6}, result)
	})
}

func TestPipelineImmediateDone(t *testing.T) {
	// Проверяем, что при мгновенном закрытии done данные не обрабатываются
	t.Run("done closed immediately", func(t *testing.T) {
		in := make(Bi)
		done := make(Bi)

		// Закрываем done сразу
		close(done)

		stage := func(in In) Out {
			out := make(Bi)
			go func() {
				defer close(out)
				for v := range in {
					out <- v.(int) * 2
				}
			}()
			return out
		}

		// Запускаем горутину для отправки данных
		go func() {
			in <- 42
			close(in)
		}()

		// Читаем результат
		result := make([]int, 0)
		for v := range ExecutePipeline(in, done, stage) {
			result = append(result, v.(int))
		}

		// Главное - не должно быть результатов
		require.Empty(t, result,
			"Should not get any results when done channel is closed immediately")
	})

	// Проверяем прерывание до начала обработки
	t.Run("done closed before sending data", func(t *testing.T) {
		in := make(Bi)
		done := make(Bi)

		// Закрываем done через короткое время
		go func() {
			time.Sleep(time.Millisecond * 10)
			close(done)
		}()

		// Ждем немного перед отправкой данных
		go func() {
			time.Sleep(time.Millisecond * 50)
			in <- 1
			in <- 2
			close(in)
		}()

		stage := func(in In) Out {
			out := make(Bi)
			go func() {
				defer close(out)
				for v := range in {
					out <- v.(int) * 2
				}
			}()
			return out
		}

		result := make([]int, 0)
		for v := range ExecutePipeline(in, done, stage) {
			result = append(result, v.(int))
		}

		// Не должно быть результатов, так как done закрыт до обработки
		require.Empty(t, result)
	})
}

func TestPipelineLargeVolume(t *testing.T) {
	// Проверяем производительность и корректность при большом объеме данных
	t.Run("process 1000 elements", func(t *testing.T) {
		in := make(Bi)
		const numElements = 1000

		// Простая стадия - инкремент
		stage := func(in In) Out {
			out := make(Bi)
			go func() {
				defer close(out)
				for v := range in {
					out <- v.(int) + 1
				}
			}()
			return out
		}

		// Отправляем данные
		go func() {
			for i := 0; i < numElements; i++ {
				in <- i
			}
			close(in)
		}()

		// Замеряем время
		start := time.Now()
		result := make([]int, 0, numElements)
		for v := range ExecutePipeline(in, nil, stage, stage, stage) { // 3 стадии
			result = append(result, v.(int))
		}
		elapsed := time.Since(start)

		// Проверяем результаты
		require.Len(t, result, numElements)
		// Каждый элемент прошел через 3 стадии +1
		for i := 0; i < numElements; i++ {
			require.Equal(t, i+3, result[i])
		}

		// Проверяем что выполнение было конкурентным
		// Последовательное выполнение: 1000 * 3 стадии = должно быть быстро
		require.Less(t, elapsed, time.Second)
	})

	// Проверяем корректное прерывание в середине обработки
	t.Run("with done signal in the middle", func(t *testing.T) {
		in := make(Bi)
		done := make(Bi)
		const numElements = 500

		var mu sync.Mutex
		processedCount := 0
		stage := func(in In) Out {
			out := make(Bi)
			go func() {
				defer close(out)
				for v := range in {
					time.Sleep(time.Microsecond * 10)
					mu.Lock()
					processedCount++
					mu.Unlock()
					out <- v
				}
			}()
			return out
		}

		// Отправляем данные
		go func() {
			for i := 0; i < numElements; i++ {
				in <- i
			}
			close(in)
		}()

		// Закрываем done после обработки части данных
		go func() {
			time.Sleep(time.Millisecond * 20)
			close(done)
		}()

		result := make([]int, 0)
		for v := range ExecutePipeline(in, done, stage, stage) {
			result = append(result, v.(int))
		}

		// Должны обработаться только некоторые элементы
		mu.Lock()
		finalCount := processedCount
		mu.Unlock()

		require.Less(t, len(result), numElements)
		require.Less(t, finalCount, numElements*2) // 2 стадии
	})
}
