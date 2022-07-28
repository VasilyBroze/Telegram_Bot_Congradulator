package entity

//СТРУКТУРА ПАРСИНГА ИЗ ГУГЛ ТАБЛИЦ
type Employee struct {
	Name       string `json:"ФИО"`
	Date       string `json:"Дата рождения"`
	Donate     string `json:"Сбор"`
	Department string `json:"Отдел"`
	Company    string `json:"Компания"`
	Male       string
	Telephone  string `json:"Телефон"`
}

//Пример метода и теста
/*
func (e Employee) getBirthdayJson(list, url string) []Employee {
	resp, _ := http.Get(fmt.Sprintf("https://tools.aimylogic.com/api/googlesheet2json?sheet=%v&id=%v", list, url))
	defer resp.Body.Close()

	employes := []Employee{}

	err := json.NewDecoder(resp.Body).Decode(&employes)
	if err != nil {
		fmt.Println(err, " body, err")
	}

	employesBirthday := []Employee{} //СТРУКТУРА ЛЮДЕЙ С ДНЁМ РОЖДЕНИЯ
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
*/
