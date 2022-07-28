package helpers

import (
	"math/rand"
	"time"
)

//СЛУЧАЙНОЕ ЧИСЛО ДЛЯ ОПРЕДЕЛЕНИЯ ТЕКСТА СООБЩЕНИЯ
func Random(max int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(max)
}
