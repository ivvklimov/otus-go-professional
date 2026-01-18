package hw04lrucache

import (
	"math/rand"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCache(t *testing.T) {
	t.Run("empty cache", func(t *testing.T) {
		c := NewCache(10)

		_, ok := c.Get("aaa")
		require.False(t, ok)

		_, ok = c.Get("bbb")
		require.False(t, ok)
	})

	t.Run("simple", func(t *testing.T) {
		c := NewCache(5)

		wasInCache := c.Set("aaa", 100)
		require.False(t, wasInCache)

		wasInCache = c.Set("bbb", 200)
		require.False(t, wasInCache)

		val, ok := c.Get("aaa")
		require.True(t, ok)
		require.Equal(t, 100, val)

		val, ok = c.Get("bbb")
		require.True(t, ok)
		require.Equal(t, 200, val)

		wasInCache = c.Set("aaa", 300)
		require.True(t, wasInCache)

		val, ok = c.Get("aaa")
		require.True(t, ok)
		require.Equal(t, 300, val)

		val, ok = c.Get("ccc")
		require.False(t, ok)
		require.Nil(t, val)
	})

	t.Run("purge logic by capacity", func(t *testing.T) {
		// n = 3, добавили 4 элемента - 1й из кэша вытолкнулся
		c := NewCache(3)

		// Добавляем 3 элемента
		c.Set("a", 1)
		c.Set("b", 2)
		c.Set("c", 3)

		// Проверяем, что все три есть
		val, ok := c.Get("a")
		require.True(t, ok)
		require.Equal(t, 1, val)

		val, ok = c.Get("b")
		require.True(t, ok)
		require.Equal(t, 2, val)

		val, ok = c.Get("c")
		require.True(t, ok)
		require.Equal(t, 3, val)

		// Добавляем 4-й элемент - должен вытолкнуть "a" (самый старый)
		c.Set("d", 4)

		// "a" должен быть вытолкнут
		val, ok = c.Get("a")
		require.False(t, ok)
		require.Nil(t, val)

		// Остальные должны остаться
		val, ok = c.Get("b")
		require.True(t, ok)
		require.Equal(t, 2, val)

		val, ok = c.Get("c")
		require.True(t, ok)
		require.Equal(t, 3, val)

		val, ok = c.Get("d")
		require.True(t, ok)
		require.Equal(t, 4, val)
	})

	t.Run("purge logic by LRU", func(t *testing.T) {
		// n = 3, добавили 3 элемента, обратились несколько раз к разным элементам
		// - добавили 4й элемент, из первой тройки вытолкнется тот элемент,
		// что был затронут наиболее давно
		c := NewCache(3)

		// Добавляем 3 элемента: a, b, c
		c.Set("a", 1)
		c.Set("b", 2)
		c.Set("c", 3)

		// Теперь история использования: [c, b, a] (a - самый старый)

		// Обращаемся к "a" - он становится самым свежим
		val, ok := c.Get("a")
		require.True(t, ok)
		require.Equal(t, 1, val)
		// Теперь: [a, c, b] (b - самый старый)

		// Обращаемся к "c" - он становится свежим
		val, ok = c.Get("c")
		require.True(t, ok)
		require.Equal(t, 3, val)
		// Теперь: [c, a, b] (b - всё ещё самый старый)

		// Обновляем значение "b" - он становится свежим
		wasInCache := c.Set("b", 22)
		require.True(t, wasInCache)
		// Теперь: [b, c, a] (a - самый старый)

		// Добавляем 4-й элемент "d"
		c.Set("d", 4)

		// Должен вытолкнуть "a" (самый давно неиспользуемый)
		val, ok = c.Get("a")
		require.False(t, ok)
		require.Nil(t, val)

		// Проверяем оставшиеся элементы
		val, ok = c.Get("b")
		require.True(t, ok)
		require.Equal(t, 22, val)

		val, ok = c.Get("c")
		require.True(t, ok)
		require.Equal(t, 3, val)

		val, ok = c.Get("d")
		require.True(t, ok)
		require.Equal(t, 4, val)
	})

	t.Run("complex LRU scenario", func(t *testing.T) {
		c := NewCache(4)

		// Заполняем кэш
		c.Set("1", "one")
		c.Set("2", "two")
		c.Set("3", "three")
		c.Set("4", "four")
		// Состояние: [4, 3, 2, 1]

		// Читаем элемент "2" - он становится свежим
		c.Get("2")
		// Состояние: [2, 4, 3, 1]

		// Обновляем элемент "3" - он становится свежим
		c.Set("3", "three-updated")
		// Состояние: [3, 2, 4, 1]

		// Добавляем новый элемент "5" - выталкивается "1" (самый старый)
		c.Set("5", "five")
		// Состояние: [5, 3, 2, 4]

		// Проверяем
		_, ok := c.Get("1")
		require.False(t, ok, "Элемент '1' должен быть вытолкнут")

		require.Equal(t, "five", getValue(t, c, "5"))
		require.Equal(t, "three-updated", getValue(t, c, "3"))
		require.Equal(t, "two", getValue(t, c, "2"))
		require.Equal(t, "four", getValue(t, c, "4"))
	})
}

// Вспомогательная функция для получения значения.
func getValue(t *testing.T, c Cache, key string) interface{} {
	t.Helper()
	val, ok := c.Get(Key(key))
	require.True(t, ok)
	return val
}

func TestCacheMultithreading(t *testing.T) {
	// t.Skip() // Remove me if task with asterisk completed.

	c := NewCache(10)
	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 1_000_000; i++ {
			c.Set(Key(strconv.Itoa(i)), i)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 1_000_000; i++ {
			c.Get(Key(strconv.Itoa(rand.Intn(1_000_000))))
		}
	}()

	wg.Wait()
	_ = t
}
