package utils

import "log/slog"

/* ----------- Estructuras de Colas ----------- */
// Definimos una estructura genérica Cola con el tipo T
type Cola[T any] struct {
	Elementos []T
}

// Enqueue agrega un elemento al final de la cola
func (c *Cola[T]) Enqueue(valor T) {
	c.Elementos = append(c.Elementos, valor)
}

// Dequeue elimina y devuelve el primer elemento de la cola
func (c *Cola[T]) Dequeue() T {
	if len(c.Elementos) == 0 {
		var zero T // Devuelve el valor "cero" del tipo T
		slog.Error("La cola está vacía")
		return zero
	}
	valor := c.Elementos[0]
	c.Elementos = c.Elementos[1:]
	return valor
}

// EstaVacia verifica si la cola está vacía
func (c *Cola[T]) EstaVacia() bool {
	return len(c.Elementos) == 0
}

/* ----------- Canales Globales ----------- */
var ChInterrupt = make(chan int)
var ChMotivoDesalojo = make(chan string)
var ChSalidaProceso = make(chan int)
var ChDumpMemory = make(chan int)
