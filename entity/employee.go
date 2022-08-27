package entity

//СТРУКТУРА ПАРСИНГА ИЗ ГУГЛ ТАБЛИЦ
type Employee struct {
	Name       string `json:"ФИО"`
	Date       string `json:"Дата рождения"`
	Donate     string `json:"Сбор"`
	Donater    string `json:"Сборщик"`
	Department string `json:"Отдел"`
	Company    string `json:"Компания"`
	Male       string
	Telephone  string `json:"Телефон"`
	Mail       string `json:"Рабочая почта"`
	Gift       string `json:"Подарок"`
}
