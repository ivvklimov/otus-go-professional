/*
ExecutePipeline — это конвейер обработки данных с возможностью аккуратно остановиться по сигналу done.

Проблема, которую нужно решить:
стадии пайплайна не знают про done и всегда пытаются передать результат дальше
(делают out <- value). Если в этот момент никто не читает выходной канал,
стадия зависает навсегда, а её goroutine не завершается.

Это критично для тестов: они ожидают, что при отмене работы
завершатся ВСЕ goroutine стадии, иначе тест зависнет по таймауту.

Как это решено:

1. Вход и выход каждой стадии оборачиваются одной функцией forward:
   forward создаёт канал и горутину, которая пересылает данные.
   При сигнале done горутина выходит из цикла, закрывает выходной канал
   и вызывает drain для входного канала, вычитывая все оставшиеся значения.

2. Drain нужен затем, чтобы ни одна стадия не осталась без читателя:
   когда forward закрывает выходной канал, стадия-отправитель может
   заблокироваться на отправке. Drain вычитывает данные и разблокирует её.

3. Порядок в defer: сначала close(out), затем drain(in):
   - close(out) мгновенно сигнализирует следующей стадии об остановке
   - drain(in) разблокирует текущую стадию, вычитывая остатки

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

func forward(in In, done In) Out {
	out := make(Bi)
	go func() {
		defer func() {
			close(out)
			drain(in) // вычитываем оставшиеся данные, чтобы разблокировать отправителя
		}()
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

func ExecutePipeline(in In, done In, stages ...Stage) Out {
	if len(stages) == 0 {
		return in
	}

	current := in
	for _, stage := range stages {
		stageIn := forward(current, done)
		stageOut := stage(stageIn)
		current = forward(stageOut, done)
	}

	return current
}
