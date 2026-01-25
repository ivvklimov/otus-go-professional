/*
ExecutePipeline — это конвейер обработки данных с возможностью аккуратно остановиться по сигналу done.

Проблема, которую нужно решить:
стадии пайплайна не знают про done и всегда пытаются передать результат дальше
(делают out <- value). Если в этот момент никто не читает выходной канал,
стадия зависает навсегда, а её goroutine не завершается.

Это критично для тестов: они ожидают, что при отмене работы
завершатся ВСЕ goroutine стадии, иначе тест зависнет по таймауту.

Как это решено:

1. Вход каждой стадии оборачивается (orDone):
   когда done закрыт, новые данные в стадию больше не поступают,
   входной канал закрывается, и стадия выходит из range in.

2. Выход каждой стадии тоже оборачивается:
   данные передаются дальше по пайплайну,
   а при закрытии done выполняется drain — вычитывание всех оставшихся значений.

3. Drain нужен затем, чтобы ни одна стадия не осталась без читателя.
   Он просто забирает все значения из канала, позволяя стадиям
   спокойно завершить работу и выйти.

Итог:
  - при отмене новые данные не принимаются,
  - все отправки в каналы разблокируются,
  - все goroutine корректно завершаются,
  - пайплайн останавливается без дедлоков и утечек.

Ключевая идея:
если goroutine что-то отправляет в канал, кто-то обязан это прочитать,
даже во время остановки пайплайна.
*/

package hw06pipelineexecution

type (
	In  = <-chan interface{}
	Out = In
	Bi  = chan interface{}
)

type Stage func(in In) (out Out)

func drain(in In) {
	for range in {
		// drain channel
		_ = struct{}{}
	}
}

func orDone(in In, done In) Out {
	out := make(Bi)
	go func() {
		defer close(out)
		for {
			select {
			case <-done:
				return
			case v, ok := <-in:
				if !ok {
					return
				}
				select {
				case <-done:
					return
				case out <- v:
				}
			}
		}
	}()
	return out
}

func forward(in In, done In) Out {
	out := make(Bi)
	go func() {
		defer close(out)
		for {
			select {
			case <-done:
				drain(in)
				return
			case v, ok := <-in:
				if !ok {
					return
				}
				select {
				case <-done:
					drain(in)
					return
				case out <- v:
				}
			}
		}
	}()
	return out
}

func ExecutePipeline(in In, done In, stages ...Stage) Out {
	if len(stages) == 0 {
		return in
	}

	current := in
	for _, stage := range stages {
		stageIn := orDone(current, done)
		stageOut := stage(stageIn)
		current = forward(stageOut, done)
	}

	return current
}
