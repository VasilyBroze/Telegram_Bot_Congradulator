/*
TODO: Создать третий пакет
TODO: Обдумать как переделать структуры с использование методов
TODO:  add .gitignore
TODO:  add users.db to .gitignore
TODO:  pkg / internal
TODO:  env / godotenv
*/
package helpers

import (
	"TelegramBot-congratulator/entity"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

//ГЕНЕРИРУЕМ СООБЩЕНИЕ ПО ГУГЛ ТАБЛИЦЕ С ЗАГОТОВКАМИ
func GetBirthdayMsg(peoples entity.Employee, list, url string) string {
	//МАССИВЫ СТРУКТУР ЧАСТЕЙ ПОЗДРАВЛЕНИЯ
	fTP, sTP, tTP := GetCongratArrays(list, url)

	var text1, text2, text3, text4, text5 string

	//ГЕНЕРИРУЕМ СЛУЧАЙНОЕ ЧИСЛО, И ПО НЕМУ ПОДСТАВЛЯЕМ ЧАСТЬ ТЕКСТА
	text1 = fTP[Random(len(fTP))].Congratulation

	//ЕСЛИ В ПЕРВОЙ ЧАСТИ УКАЗАН ПОЛ, ПРОВЕРЯЕМ ПОЛ СОТРУДНИКА
	for strings.HasSuffix(text1, " *Ж") && peoples.Male == "М" {
		text1 = fTP[Random(len(fTP))].Congratulation
		time.Sleep(77 * time.Microsecond)
	}
	for strings.HasSuffix(text1, " *М") && peoples.Male == "Ж" {
		text1 = fTP[Random(len(fTP))].Congratulation
		time.Sleep(77 * time.Microsecond)
	}
	//ЕСЛИ ПОЛ ОПРЕДЕЛИТЬ НЕ УДАЛОСЬ - НЕ ИСПОЛЬЗУЕМ НАЧАЛЬНЫЕ ФРАЗЫ В КОТОРЫХ ОН УКАЗАН
	for peoples.Male == "?" && (strings.HasSuffix(text1, " *М") || strings.HasSuffix(text1, " *Ж")) {
		text1 = fTP[Random(len(fTP))].Congratulation
		time.Sleep(77 * time.Microsecond)
	}

	//УДАЛЯЕМ УКАЗАТЕЛИ ПОЛА В НАЧАЛЬНОЙ ФРАЗЕ
	if strings.HasSuffix(text1, " *Ж") {
		text1 = strings.Replace(text1, " *Ж", "", 1)
	}
	if strings.HasSuffix(text1, " *М") {
		text1 = strings.Replace(text1, " *М", "", 1)
	}
	//ПОЛУЧАЕМ ИМЯ В НУЖНОМ ПАДЕЖЕ В ЗАВИСИМОСТИ ОТ УКАЗАТЕЛЯ ПАДЕЖА В ГУГЛ ТАБЛИЦЕ
	if strings.HasSuffix(text1, " *В") {
		text1 = strings.Replace(text1, " *В", "", 1)
		peoples.Name = GetPrettySuffix(peoples.Name, "V")
	}
	if strings.HasSuffix(text1, " *Д") {
		text1 = strings.Replace(text1, " *Д", "", 1)
		peoples.Name = GetPrettySuffix(peoples.Name, "D")
	}
	if strings.HasSuffix(text1, " *Р") {
		text1 = strings.Replace(text1, " *Р", "", 1)
		peoples.Name = GetPrettySuffix(peoples.Name, "R")
	}

	//ПОЛУЧАЕМ ОТДЕЛ В НУЖНОМ ПАДЕЖЕ
	peoples.Department = GetPrettySuffix(peoples.Department, "R")

	text2 = sTP[Random(len(sTP))].WishYou
	//ПОЖЕЛАНИЯ ГЕНЕРИРУЕМ ТАК, ЧТОБЫ ОНИ НЕ ПОВТОРЯЛИСЬ
	text3 = tTP[Random(len(tTP))].Sentiments
	for text4 == "" || text4 == text3 {
		text4 = tTP[Random(len(tTP))].Sentiments
	}
	for text5 == "" || text5 == text4 || text5 == text3 {
		text5 = tTP[Random(len(tTP))].Sentiments
	}
	msg := fmt.Sprintf("%v %v из %v! %v %v, %v и %v!", text1, peoples.Name, peoples.Department, text2, text3, text4, text5)
	return msg
}

//ПАРСИМ ТАБЛИЦУ С ТЕКСТОМ ПОЗДРАВЛЕНИЙ И РАСПРЕДЕЛЯЕМ ИХ ПО МАССИВАМ
func GetCongratArrays(list, url string) ([]entity.TextFirstPart, []entity.TextSecondPart, []entity.TextThirdPart) {
	resp, _ := http.Get(fmt.Sprintf("https://tools.aimylogic.com/api/googlesheet2json?sheet=%v&id=%v", list, url))
	defer resp.Body.Close()

	//МАССИВЫ ДЛЯ ПАРСИНГА
	fTP := []entity.TextFirstPart{}
	sTP := []entity.TextSecondPart{}
	tTP := []entity.TextThirdPart{}

	fTPraw := []entity.TextFirstPart{}
	sTPraw := []entity.TextSecondPart{}
	tTPraw := []entity.TextThirdPart{}

	body, err := ioutil.ReadAll(resp.Body) //ПОЛУЧИЛИ JSON
	if err != nil {
		panic(err)
	}

	if err := json.Unmarshal(body, &fTP); err != nil {
		fmt.Println(err)
		panic(err)
	}

	if err := json.Unmarshal(body, &sTP); err != nil {
		fmt.Println(err)
	}

	if err := json.Unmarshal(body, &tTP); err != nil {
		fmt.Println(err)
	}

	//ФИЛЬТРУЕМ ПУСТЫЕ СТРОКИ
	for _, first := range fTP {
		if first.Congratulation != "" {
			fTPraw = append(fTPraw, first)
		}
	}

	for _, second := range sTP {
		if second.WishYou != "" {
			sTPraw = append(sTPraw, second)
		}
	}

	for _, third := range tTP {
		if third.Sentiments != "" {
			tTPraw = append(tTPraw, third)
		}
	}

	return fTPraw, sTPraw, tTPraw
}

//ПОЛУЧАЕМ ИМЯ В НУЖНОМ ПАДЕЖЕ
func GetPrettySuffix(people, padej string) string {
	name := people
	people = strings.Replace(people, " ", "%20", -1)
	resp, err := http.Get(fmt.Sprint("http://ws3.morpher.ru/russian/declension?s=" + people + "&format=json"))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	rodSuffix := entity.RightSuffix{}

	body, err := ioutil.ReadAll(resp.Body) //ПОЛУЧИЛИ JSON
	if err != nil {
		panic(err)
	}

	if err := json.Unmarshal(body, &rodSuffix); err != nil {
		fmt.Println(err)
	}

	//ЕСЛИ НЕ ПОЛУЧИЛИ ИМЯ В НУЖНОМ ПАДЕЖЕ - ВОЗВРАЩАЕМ КАК ЕСТЬ
	if rodSuffix.Code != 0 {
		name = strings.Replace(people, "%20", " ", -1)
		fmt.Println("ОШИБКА СЕРВИСА ПАДЕЖЕЙ")
		return name
	}

	switch padej {
	case "V":
		name = rodSuffix.NameV
	case "D":
		name = rodSuffix.NameD
	case "R":
		name = rodSuffix.NameR
	}

	return name
}

//ПАРСИМ ЛЮДЕЙ У КОТОРЫЕ МОГУТ БЫТЬ СБОРЩИКАМИ СРЕДСТВ
func GetDonationListJson(list, url string) []entity.Employee {
	resp, _ := http.Get(fmt.Sprintf("https://tools.aimylogic.com/api/googlesheet2json?sheet=%v&id=%v", list, url))
	defer resp.Body.Close()

	employes := []entity.Employee{}

	err := json.NewDecoder(resp.Body).Decode(&employes)
	if err != nil {
		fmt.Println(err, " body, err")
	}

	employesBirthday := []entity.Employee{} //СТРУКТУРА ЛЮДЕЙ С ДНЁМ РОЖДЕНИЯ
	currentTime := time.Now()
	tomorrow := currentTime.Add(24 * time.Hour)

	var strMonth, strDay, strDate string

	//КОНВЕРТАЦИЯ МЕСЯЦА
	switch int(tomorrow.Month()) {
	case 1:
		strMonth = "янв."
	case 2:
		strMonth = "февр"
	case 3:
		strMonth = "мар."
	case 4:
		strMonth = "апр."
	case 5:
		strMonth = "мая"
	case 6:
		strMonth = "июн."
	case 7:
		strMonth = "июл."
	case 8:
		strMonth = "авг."
	case 9:
		strMonth = "сент."
	case 10:
		strMonth = "окт."
	case 11:
		strMonth = "нояб."
	case 12:
		strMonth = "дек."

	}

	//КОНВЕРТАЦИЯ ДНЯ
	strDay = strconv.Itoa(tomorrow.Day())

	strDate = strDay + " " + strMonth //ПРИВОДИМ ДАТУ К ВИДУ ГУГЛДОК

	//В ЦИКЛЕ ПО ВСЕМ ЛЮДЯМ ИЩЕМ ТЕХ КТО МОЖЕТ СОБИРАТЬ СРЕДСТВА И У КОГО ЗАВТРА НЕ ДЕНЬ РОЖДЕНИЯ И ДОБАВЛЯЕМ ИХ В НОВУЮ СТРУКТУРУ
	for _, empl := range employes {
		if strings.HasPrefix(empl.Date, strDate) == false {

			//ИЗМЕНЯЕМ НАЗВАНИЕ ОТДЕЛА НА БОЛЕЕ КОРОТКОЕ
			switch {
			case strings.Contains(empl.Department, "ПТО"):
				empl.Department = "Отдел ПТО"
			case strings.Contains(empl.Department, "(ПО)"):
				empl.Department = "Отдел IT"
			case strings.Contains(empl.Department, "ПНР"):
				empl.Department = "Отдел ПНР"
			case strings.Contains(empl.Department, "("): //СОКРАЩАЕМ НАЗВАНИЕ ОТДЕЛА ДО ПЕРВОЙ СКОБКИ
				dep := strings.Split(empl.Department, "(")
				if len(dep) > 0 {
					empl.Department = dep[0]
				}
			}
			if empl.Donate == "Да" {
				employesBirthday = append(employesBirthday, empl)
			}
		}
	}
	return employesBirthday
}

//ПАРСИМ ЛЮДЕЙ У КОТОРЫХ ЗАВТРА ДЕНЬ РОЖДЕНИЯ, ОПРЕДЕЛЯЕМ ПОЛ
func GetAnonceBirthdayJson(list, url string) []entity.Employee {
	resp, _ := http.Get(fmt.Sprintf("https://tools.aimylogic.com/api/googlesheet2json?sheet=%v&id=%v", list, url))
	defer resp.Body.Close()

	employes := []entity.Employee{}

	err := json.NewDecoder(resp.Body).Decode(&employes)
	if err != nil {
		fmt.Println(err, " body, err")
	}

	employesBirthday := []entity.Employee{} //СТРУКТУРА ЛЮДЕЙ С ДНЁМ РОЖДЕНИЯ
	currentTime := time.Now()
	tomorrow := currentTime.Add(24 * time.Hour)

	var strMonth, strDay, strDate string

	//КОНВЕРТАЦИЯ МЕСЯЦА
	switch int(tomorrow.Month()) {
	case 1:
		strMonth = "янв."
	case 2:
		strMonth = "февр"
	case 3:
		strMonth = "мар."
	case 4:
		strMonth = "апр."
	case 5:
		strMonth = "мая"
	case 6:
		strMonth = "июн."
	case 7:
		strMonth = "июл."
	case 8:
		strMonth = "авг."
	case 9:
		strMonth = "сент."
	case 10:
		strMonth = "окт."
	case 11:
		strMonth = "нояб."
	case 12:
		strMonth = "дек."

	}

	//КОНВЕРТАЦИЯ ДНЯ
	strDay = strconv.Itoa(tomorrow.Day())

	strDate = strDay + " " + strMonth //ПРИВОДИМ ДАТУ К ВИДУ ГУГЛДОК

	//В ЦИКЛЕ ПО ВСЕМ ЛЮДЯМ ИЩЕМ ТЕХ У КОГО ЗАВТРА ДЕНЬ РОЖДЕНИЯ И ДОБАВЛЯЕМ ИХ В НОВУЮ СТРУКТУРУ
	for _, empl := range employes {
		//ЕСЛИ ДАТА РОЖДЕНИЯ СОВПАДАЕТ С ЗАВТРАШНИМ ДНЁМ, ЧЕЛОВЕК ХОЧЕТ ПОЛУЧАТЬ ПОДАРКИ И СОСТОИТ В КОМПАНИИ ЛИИС/СИМПЛЧАРЖ ДОБАВЛЯЕМ ЕГО В СТРУКТУРУ
		if strings.HasPrefix(empl.Date, strDate) && empl.Gift != "Нет" && (strings.HasPrefix(empl.Company, "ЛИИС") || strings.HasPrefix(empl.Company, "Симпл")) {
			shortName := strings.Split(empl.Name, " ")
			//ЕСЛИ ФИО ИЗ 3 СЛОВ - ОПРЕДЕЛЯЕМ ПОЛ ПО ОТЧЕСТВУ, УБИРАЕМ ОТЧЕСТВО
			if len(shortName) == 3 {
				switch {
				case
					strings.HasSuffix(shortName[2], "ч"):
					empl.Male = "М"
				case
					strings.HasSuffix(shortName[2], "а"):
					empl.Male = "Ж"
				default:
					empl.Male = "?"
				}
				empl.Name = shortName[1] + " " + shortName[0]
			} else {
				empl.Male = "?"
			}

			//ИЗМЕНЯЕМ НАЗВАНИЕ ОТДЕЛА НА БОЛЕЕ КОРОТКОЕ
			switch {
			case strings.Contains(empl.Department, "ПТО"):
				empl.Department = "Отдел ПТО"
			case strings.Contains(empl.Department, "(ПО)"):
				empl.Department = "Отдел IT"
			case strings.Contains(empl.Department, "ПНР"):
				empl.Department = "Отдел ПНР"
			case strings.Contains(empl.Department, "("): //СОКРАЩАЕМ НАЗВАНИЕ ОТДЕЛА ДО ПЕРВОЙ СКОБКИ
				dep := strings.Split(empl.Department, "(")
				if len(dep) > 0 {
					empl.Department = dep[0]
				}
			}

			employesBirthday = append(employesBirthday, empl)
		}
	}
	return employesBirthday
}

//ПАРСИМ ЛЮДЕЙ У КОТОРЫХ СЕГОДНЯ ДЕНЬ РОЖДЕНИЯ, ОПРЕДЕЛЯЕМ ПОЛ
func GetBirthdayJson(list, url string) []entity.Employee {
	resp, _ := http.Get(fmt.Sprintf("https://tools.aimylogic.com/api/googlesheet2json?sheet=%v&id=%v", list, url))
	defer resp.Body.Close()

	employes := []entity.Employee{}

	err := json.NewDecoder(resp.Body).Decode(&employes)
	if err != nil {
		fmt.Println(err, " body, err")
	}

	employesBirthday := []entity.Employee{} //СТРУКТУРА ЛЮДЕЙ С ДНЁМ РОЖДЕНИЯ
	currentTime := time.Now()

	var strMonth, strDay, strDate string

	//КОНВЕРТАЦИЯ МЕСЯЦА
	switch int(currentTime.Month()) {
	case 1:
		strMonth = "янв."
	case 2:
		strMonth = "февр"
	case 3:
		strMonth = "мар."
	case 4:
		strMonth = "апр."
	case 5:
		strMonth = "мая"
	case 6:
		strMonth = "июн."
	case 7:
		strMonth = "июл."
	case 8:
		strMonth = "авг."
	case 9:
		strMonth = "сент."
	case 10:
		strMonth = "окт."
	case 11:
		strMonth = "нояб."
	case 12:
		strMonth = "дек."

	}

	//КОНВЕРТАЦИЯ ДНЯ
	strDay = strconv.Itoa(currentTime.Day())

	strDate = strDay + " " + strMonth //ПРИВОДИМ ДАТУ К ВИДУ ГУГЛДОК

	//В ЦИКЛЕ ПО ВСЕМ ЛЮДЯМ ИЩЕМ ТЕХ У КОГО ДЕНЬ РОЖДЕНИЯ И ДОБАВЛЯЕМ ИХ В НОВУЮ СТРУКТУРУ
	for _, empl := range employes {
		if strings.HasPrefix(empl.Date, strDate) && (strings.HasPrefix(empl.Company, "ЛИИС") || strings.HasPrefix(empl.Company, "Симпл")) {
			shortName := strings.Split(empl.Name, " ")
			//ЕСЛИ ФИО ИЗ 3 СЛОВ - ОПРЕДЕЛЯЕМ ПОЛ ПО ОТЧЕСТВУ, УБИРАЕМ ОТЧЕСТВО
			if len(shortName) == 3 {
				switch {
				case
					strings.HasSuffix(shortName[2], "ч"):
					empl.Male = "М"
				case
					strings.HasSuffix(shortName[2], "а"):
					empl.Male = "Ж"
				default:
					empl.Male = "?"
				}
				empl.Name = shortName[1] + " " + shortName[0]
			} else {
				empl.Male = "?"
			}

			//ИЗМЕНЯЕМ НАЗВАНИЕ ОТДЕЛА НА БОЛЕЕ КОРОТКОЕ
			switch {
			case strings.Contains(empl.Department, "ПТО"):
				empl.Department = "Отдел ПТО"
			case strings.Contains(empl.Department, "(ПО)"):
				empl.Department = "Отдел IT"
			case strings.Contains(empl.Department, "ПНР"):
				empl.Department = "Отдел ПНР"
			case strings.Contains(empl.Department, "("): //СОКРАЩАЕМ НАЗВАНИЕ ОТДЕЛА ДО ПЕРВОЙ СКОБКИ
				dep := strings.Split(empl.Department, "(")
				if len(dep) > 0 {
					empl.Department = dep[0]
				}
			}

			employesBirthday = append(employesBirthday, empl)
		}
	}
	return employesBirthday
}

//ПАРСИМ ЛЮДЕЙ У КОТОРЫХ ЗАВТРА ДЕНЬ РОЖДЕНИЯ, ОПРЕДЕЛЯЕМ ПОЛ
func GetBirthdayMonthListJson(list, url, month string) []entity.Employee {
	resp, _ := http.Get(fmt.Sprintf("https://tools.aimylogic.com/api/googlesheet2json?sheet=%v&id=%v", list, url))
	defer resp.Body.Close()

	employes := []entity.Employee{}

	err := json.NewDecoder(resp.Body).Decode(&employes)
	if err != nil {
		fmt.Println(err, " body, err")
	}
	//СТРУКТУРА ЛЮДЕЙ С ДНЁМ РОЖДЕНИЯ В КОНКРЕТНОМ МЕСЯЦЕ
	employesBirthday := []entity.Employee{}

	var strMonth string

	//УКОРАЧИВАЕМ МЕСЯЦ ДО 3х БУКВ ЕСЛИ ОН ДЛИННЕЕ
	if len(month) >= 6 {
		month = month[0:6]
	}

	//КОНВЕРТАЦИЯ МЕСЯЦА
	switch strings.ToLower(month) {
	case "янв", "1":
		strMonth = "янв."
	case "фев", "2":
		strMonth = "февр"
	case "мар", "3":
		strMonth = "мар."
	case "апр", "4":
		strMonth = "апр."
	case "май", "5":
		strMonth = "мая"
	case "июн", "6":
		strMonth = "июн."
	case "июл", "7":
		strMonth = "июл."
	case "авг", "8":
		strMonth = "авг."
	case "сен", "9":
		strMonth = "сент."
	case "окт", "10":
		strMonth = "окт."
	case "ноя", "11":
		strMonth = "нояб."
	case "дек", "12":
		strMonth = "дек."
	default:
		return nil
	}

	//В ЦИКЛЕ ПО ВСЕМ ЛЮДЯМ ИЩЕМ ТЕХ У КОГО ДЕНЬ РОЖДЕНИЯ В УКАЗАННОМ МЕСЯЦЕ И ДОБАВЛЯЕМ ИХ В НОВУЮ СТРУКТУРУ
	for _, empl := range employes {
		if strings.Contains(empl.Date, strMonth) && (strings.HasPrefix(empl.Company, "ЛИИС") || strings.HasPrefix(empl.Company, "Симпл")) {
			shortName := strings.Split(empl.Name, " ")
			//ЕСЛИ ФИО ИЗ 3 СЛОВ - ОПРЕДЕЛЯЕМ ПОЛ ПО ОТЧЕСТВУ, УБИРАЕМ ОТЧЕСТВО
			if len(shortName) == 3 {
				switch {
				case
					strings.HasSuffix(shortName[2], "ч"):
					empl.Male = "М"
				case
					strings.HasSuffix(shortName[2], "а"):
					empl.Male = "Ж"
				default:
					empl.Male = "?"
				}
				empl.Name = shortName[1] + " " + shortName[0]
			} else {
				empl.Male = "?"
			}

			//ИЗМЕНЯЕМ НАЗВАНИЕ ОТДЕЛА НА БОЛЕЕ КОРОТКОЕ
			switch {
			case strings.Contains(empl.Department, "ПТО"):
				empl.Department = "Отдел ПТО"
			case strings.Contains(empl.Department, "(ПО)"):
				empl.Department = "Отдел IT"
			case strings.Contains(empl.Department, "ПНР"):
				empl.Department = "Отдел ПНР"
			case strings.Contains(empl.Department, "("): //СОКРАЩАЕМ НАЗВАНИЕ ОТДЕЛА ДО ПЕРВОЙ СКОБКИ
				dep := strings.Split(empl.Department, "(")
				if len(dep) > 0 {
					empl.Department = dep[0]
				}
			}

			employesBirthday = append(employesBirthday, empl)
		}
	}
	return employesBirthday
}

//ФУНКЦИЯ ПОЛУЧЕНИЯ НАСТРОЕК БОТА
func GetSettings(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(f)
}
