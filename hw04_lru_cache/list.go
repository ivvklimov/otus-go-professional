package hw04lrucache

// List интерфейс двусвязного списка.
type List interface {
	Len() int
	Front() *ListItem
	Back() *ListItem
	PushFront(v interface{}) *ListItem
	PushBack(v interface{}) *ListItem
	Remove(i *ListItem)
	MoveToFront(i *ListItem)
}

// ListItem - элемент двусвязного списка.
type ListItem struct {
	Value interface{}
	Next  *ListItem
	Prev  *ListItem
}

// list - внутренняя реализация двусвязного списка.
type list struct {
	front *ListItem
	back  *ListItem
	len   int
}

// PushFront добавляет элемент в начало списка.
func (l *list) PushFront(v interface{}) *ListItem {
	item := &ListItem{Value: v}

	if l.front == nil {
		l.front = item
		l.back = item
	} else {
		item.Next = l.front
		l.front.Prev = item
		l.front = item
	}

	l.len++
	return item
}

// PushBack добавляет элемент в конец списка.
func (l *list) PushBack(v interface{}) *ListItem {
	item := &ListItem{Value: v}

	if l.back == nil {
		l.front = item
		l.back = item
	} else {
		item.Prev = l.back
		l.back.Next = item
		l.back = item
	}

	l.len++
	return item
}

// Remove удаляет элемент из списка.
func (l *list) Remove(i *ListItem) {
	if i == nil {
		return
	}

	if i.Prev != nil {
		i.Prev.Next = i.Next
	} else {
		l.front = i.Next
	}

	if i.Next != nil {
		i.Next.Prev = i.Prev
	} else {
		l.back = i.Prev
	}

	i.Next = nil
	i.Prev = nil
	l.len--
}

// MoveToFront перемещает элемент в начало списка.
func (l *list) MoveToFront(i *ListItem) {
	// Проверяем, что элемент существует, список не пуст, и элемент не первый
	if i == nil || l.front == nil || i == l.front {
		return
	}

	// Исключаем i из списка: предыдущий элемент теперь ссылается на следующий за i
	i.Prev.Next = i.Next

	if i.Next != nil {
		i.Next.Prev = i.Prev
	} else {
		l.back = i.Prev
	}

	// Вставляем i в начало
	i.Prev = nil
	i.Next = l.front
	l.front.Prev = i
	l.front = i
}

// Len возвращает количество элементов в списке.
func (l *list) Len() int {
	return l.len
}

// Front возвращает первый элемент списка.
func (l *list) Front() *ListItem {
	return l.front
}

// Back возвращает последний элемент списка.
func (l *list) Back() *ListItem {
	return l.back
}

// NewList создает и возвращает новый двусвязный список.
func NewList() List {
	return new(list)
}
