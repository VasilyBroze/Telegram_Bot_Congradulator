package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

//ИМЯ В НУЖНОМ ПАДЕЖЕ
type rSuffix struct {
	NameV string `json:"В"`
	NameD string `json:"Д"`
	NameR string `json:"Р"`
	Code  int    `json:"code"`
}

//СТРУКТУРА ПАРСИНГА ИЗ ПОЖЕЛАНИЙ
type TextFirstPart struct {
	Congratulation string `json:"Поздравляю"`
}

//СТРУКТУРА ПАРСИНГА ИЗ ПОЖЕЛАНИЙ
type TextSecondPart struct {
	WishYou string `json:"Желаю"`
}

//СТРУКТУРА ПАРСИНГА ИЗ ПОЖЕЛАНИЙ
type TextThirdPart struct {
	Sentiments string `json:"Пожелание"`
}

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

//СТРУКТУРА ПОДПИСЧИКОВ
type UsersForSpam struct {
	ChatID int64
	Name   string
}

//НАСТРОЙКИ БОТА
type BotSettings struct {
	Google_sheet_bday_url  string `json:"google_bday_url"`
	Google_sheet_bday_list string `json:"google_bday_list"`
	Google_sheet_text_url  string `json:"google_text_url"`
	Google_sheet_text_list string `json:"google_text_list"`
	Bot_token              string `json:"bot_token"`
	Chat_id                int64  `json:"chat_id"`
	Anonce_pass            string `json:"anonce_password"`
}

func main() {

	var BotSets BotSettings

	//СОЗДАНИЕ БД
	database, _ := sql.Open("sqlite3", "./users.db")
	statement, _ := database.Prepare("CREATE TABLE IF NOT EXISTS people (id INTEGER PRIMARY KEY, chat_id INTEGER,username TEXT)")
	statement.Exec()
	//БД СОЗДАНА

	//ИЗВЛЕКАЕМ ИЗ ФАЙЛА С НАСТРОЙКАМИ ПОЛЯ
	bs, err := getSettings("settings.json")
	if err != nil {
		fmt.Println("open file error: " + err.Error())
		return
	}

	if err := json.Unmarshal(bs, &BotSets); err != nil {
		fmt.Println(err)
		return
	}

	bot, err := tgbotapi.NewBotAPI(BotSets.Bot_token) //БОТ ПОЗДРАВЛЯТОР ЛИИСОВИЧ
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	//СПАМ В ЛИЧКУ ПОДПИСАВШИМСЯ
	go func() {
		u := []UsersForSpam{}
		a := UsersForSpam{}
		rows, _ := database.Query("SELECT chat_id, username FROM people")

		//ПРОВЕРЯЕМ ВСЕ ДАННЫЕ В ТАБЛИЦЕ ПО ЧАТ ID
		for rows.Next() {
			rows.Scan(&a.ChatID, &a.Name)
			u = append(u, a)
		}
		for {
			//АНОНС ТОЛЬКО В ПЕРИОД 10-11
			currentTime := time.Now()

			if currentTime.Hour() == 14 {
				//ПОЛУЧАЕМ СПИСОК ЛЮДЕЙ У КОГО ДР ЗАВТРА
				birthdayTomorrow := getAnonceBirthdayJson(BotSets.Google_sheet_bday_list, BotSets.Google_sheet_bday_url)
				//ЕСЛИ ЛЮДЕЙ У КОТОРЫЙ ДР ЗАВТРА ХОТЯБЫ 1 ДЕЛАЕМ РАССЫЛКУ
				if len(birthdayTomorrow) > 0 {
					//ПОЛУЧАЕМ СПИСОК ЛЮДЕЙ КОТОРЫЕ ГОТОВЫ СОБИРАТЬ ДЕНЬГИ
					donatorList := getDonationListJson(BotSets.Google_sheet_bday_list, BotSets.Google_sheet_bday_url)
					//ИТЕРИРУЕМСЯ ПО ЛЮДЯМ У КОТОРЫХ ДР
					for _, peoples := range birthdayTomorrow {

						//СКЛОНЯЕМ ИМЯ И ОТДЕЛ
						nameR := getPrettySuffix(peoples.Name, "R")
						departmentR := getPrettySuffix(peoples.Department, "R")
						//ИЩЕМ ЧЕЛОВЕКА ОТВЕТСТВЕННОГО ЗА СБОР СРЕДСТВ ВНУТРИ ОТДЕЛА
						var myDonator Employee
						for _, donator := range donatorList {
							if donator.Department == peoples.Department && donator.Name != peoples.Name {
								myDonator = donator
							}
						}
						//ЕСЛИ НЕ НАШЛИ КОМУ ПЕРЕВОДИТЬ ИЗ ОТДЕЛА, ИЩЕМ В ОТДЕЛЕ HR
						if myDonator.Name == "" {
							for _, donator := range donatorList {
								if donator.Department == "Отдел по работе с персоналом" && donator.Name != peoples.Name {
									myDonator = donator
								}
							}
						}

						//РАССЫЛАЕМ ПОДПИСЧИКАМ ИЗ БД
						for _, follower := range u {
							if peoples.Name != follower.Name {
								if departmentR != "" {
									msg := fmt.Sprintf("Завтра день рождения у %s из %s!\nПодарок собирает %s.\nПринимает переводы по номеру %v\nhttps://web3.online.sberbank.ru/transfers/client", nameR, departmentR, myDonator.Name, myDonator.Telephone)
									bot.Send(tgbotapi.NewMessage(follower.ChatID, msg))
								} else {
									msg := fmt.Sprintf("Завтра день рождения у %s!\nПодарок собирает %s.\nПринимает переводы по номеру %v\nhttps://web3.online.sberbank.ru/transfers/client", nameR, departmentR, myDonator.Name, myDonator.Telephone)
									bot.Send(tgbotapi.NewMessage(follower.ChatID, msg))
								}
							}
						}

						time.Sleep(1 * time.Minute) //minute
					}
				}
			}
			time.Sleep(1 * time.Hour) //hour
		}
	}()

	//НАСТРОЙКА СЛУШАТЕЛЯ
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)
	//ВЫПОЛНЕНИЕ КОМАНД ПОЛЬЗОВАТЕЛЯ
	for update := range updates {
		if update.Message == nil { // If we got a message
			continue
		}
		command := strings.Split(update.Message.Text, " ")
		command[0] = strings.ToUpper(command[0])
		switch command[0] {

		case "РЕГИСТРАЦИЯ": //ДОБАВИТЬ ЧЕЛОВЕКА В БД
			if len(command) != 3 {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Не понял Вас, Жду сообщения вида Регистация Иван Иванов"))
			} else {
				//ИМЯ ПОЛЬЗОВАТЕЛЯ
				userInputName := command[1] + " " + command[2]

				//СЧИТЫВАНИЕ ИЗ БАЗЫ
				data1, _ := database.Query("SELECT chat_id, username FROM people WHERE chat_id = ?", update.Message.Chat.ID)
				var chatId float64
				var username string

				data1.Next()
				data1.Scan(&chatId, &username)
				data1.Close()
				if chatId == 0 {
					//ЕСЛИ СТРОКИ НЕТ - ДОБАВЛЕНИЕ СТРОКИ
					statement, _ = database.Prepare("INSERT INTO people (chat_id, username) VALUES (?, ?)")
					statement.Exec(update.Message.Chat.ID, userInputName)
					//ВЫВОД В ЧАТ
					regComplited := fmt.Sprintf("Регистрация завершена, %v", userInputName)
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, regComplited))

				} else {
					//ЕСЛИ СТРОКА ЕСТЬ - ОБНОВЛЯЕМ ЗНАЧЕНИЕ
					_, err := database.Exec("UPDATE people SET username=? WHERE chat_id = ?", userInputName, update.Message.Chat.ID)
					if err != nil {
						fmt.Println(err)
					}
					//ВЫВОД В ЧАТ
					regUpdated := fmt.Sprintf("Имя изменено на %v", userInputName)
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, regUpdated))
				}
			}

		case "УДАЛИТЬСЯ": //УДАЛИТЬСЯ ИЗ БД

			//СЧИТЫВАНИЕ ИЗ БАЗЫ
			data1, _ := database.Query("SELECT chat_id FROM people WHERE chat_id = ?", update.Message.Chat.ID)
			var chatId int

			data1.Next()
			data1.Scan(&chatId)
			data1.Close()
			if chatId == 0 {
				//ЕСЛИ СТРОКИ ВЫВОДИМ СООБЩЕНИЕ В ЧАТ

				delFailed := fmt.Sprintf("Вас не было в рассылке")
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, delFailed))

			} else {
				//ЕСЛИ СТРОКА ЕСТЬ - УДАЛЯЕМ ПОЛЬЗОВАТЕЛЯ
				_, err := database.Exec("DELETE FROM people WHERE chat_id = ?", update.Message.Chat.ID)
				if err != nil {
					fmt.Println("Ошибка удаления")
				}
				//ВЫВОД В ЧАТ
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Вы удалены из рассылки"))
			}

		case "СТАТУС":

			//СЧИТЫВАНИЕ ИЗ БАЗЫ
			data1, _ := database.Query("SELECT chat_id, username FROM people WHERE chat_id = ?", update.Message.Chat.ID)
			var chatId int
			var username string

			data1.Next()
			data1.Scan(&chatId, &username)
			data1.Close()
			if username != "" {
				statusMsg := fmt.Sprintf("Ваше имя в рассылке %v", username)
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, statusMsg))
			} else {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Вас нет в рассылке :("))
			}

		case "/DESCRIPTION", "ОПИСАНИЕ":
			msg := fmt.Sprintf("Описание комманд:\nРегистация Имя Фамилия - зарегистрироваться или обновить данные\nУдалиться - удалиться из рассылок\nСтатус - ваше имя в рассылке\nПодписчики - список подписавшихся")
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))

		case "/START":
			msg := fmt.Sprintf("Привет! Я помогу тебе поздравлять твоих коллег без миллионов надоедливых чатов :-)\n" +
				"Для начала, зарегистрируйся. Примерно так: \nРегистрация Иван Иванов (сначала имя, потом фамилия)\n" +
				"Чтобы узнать что я умею введи Описание\n" +
				"Хорошего тебе дня!")
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))

		case "ОБЪЯВЛЕНИЕВСЕМ":
			if len(command) > 1 {
				if command[1] == BotSets.Anonce_pass {
					trim := fmt.Sprintf("ОБЪЯВЛЕНИЕВСЕМ %v ", BotSets.Anonce_pass)
					msg := strings.TrimPrefix(update.Message.Text, trim)

					rows, _ := database.Query("SELECT chat_id FROM people")
					var chatID int64

					//ПОЛУЧАЕМ СПИСОК ЧАТОВ ПОДПИСЧИКОВ И РАССЫЛАЕМ СООБЩЕНИЕ
					for rows.Next() {
						rows.Scan(&chatID)
						bot.Send(tgbotapi.NewMessage(chatID, msg))
					}

				} else {
					//ЕСЛИ ПАРОЛЬ НЕВЕРНЫЙ
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Неверный пароль, попробуйте снова"))
				}
			} else {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Неверная команда, Введите ОБЪЯВЛЕНИЕ (Пароль) Текст"))
			}

		case "ОБЪЯВЛЕНИЕКРОМЕ":
			if len(command) > 4 {
				if command[1] == BotSets.Anonce_pass {
					//ИМЯ ЧЕЛОВЕКА КОТОРОМУ СООБЩЕНИЕ НЕ ПОЛЕТИТ
					ignoreName := command[2] + " " + command[3]
					trim := fmt.Sprintf("ОБЪЯВЛЕНИЕКРОМЕ %v %v ", BotSets.Anonce_pass, ignoreName)
					msg := strings.TrimPrefix(update.Message.Text, trim)

					rows, _ := database.Query("SELECT chat_id, username FROM people")
					var chatID int64
					var userName string

					//ПОЛУЧАЕМ СПИСОК ЧАТОВ ПОДПИСЧИКОВ И РАССЫЛАЕМ СООБЩЕНИЕ
					for rows.Next() {
						rows.Scan(&chatID, &userName)
						if userName != ignoreName {
							bot.Send(tgbotapi.NewMessage(chatID, msg))
						}
					}

				} else {
					//ЕСЛИ ПАРОЛЬ НЕВЕРНЫЙ
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Неверный пароль, попробуйте снова"))
				}
			} else {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Неверная команда, Введите ОБЪЯВЛЕНИЕ (Пароль) Текст"))
			}

		case "ПОДПИСЧИКИ":
			msg := ""
			var sum int

			rows, _ := database.Query("SELECT username FROM people")
			var followers string

			//ПРОВЕРЯЕМ ВСЕ ДАННЫЕ В ТАБЛИЦЕ
			for rows.Next() {
				rows.Scan(&followers)

				sum += 1
				if msg != "" {
					msg += fmt.Sprintf(", %s", followers)
				} else {
					msg += fmt.Sprintf("%s", followers)
				}
			}
			msg += fmt.Sprintf("\nЧисло подписчиков %v", sum)
			//ВЫВОД В ЧАТ
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))

		default:
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Команда не найдена, посмотрите Описание"))
		}
	}
	//КОМАНДЫ КОНЧИЛАСЬ

	/*//СПАМ В ОБЩИЙ ЧАТ
	go func() {
		for {
			//ПОЗДРАВЛЕНИЕ ТОЛЬКО В ПЕРИОД 10-11
			currentTime := time.Now()

			if currentTime.Hour() == 9 {

				birthdayToday := getBirthdayJson(BotSets.Google_sheet_bday_list, BotSets.Google_sheet_bday_url)

				if len(birthdayToday) > 0 {

					for _, peoples := range birthdayToday {
						fmt.Println(peoples)
						msg := getBirthdayMsg(peoples, BotSets.Google_sheet_text_list, BotSets.Google_sheet_text_url)
						bot.Send(tgbotapi.NewMessage(BotSets.Chat_id, msg)) //ОТПРАВИТЬ В ТЕСТОВЫЙ ЧАТ
						//(678187421 личный чат)(-728590508 тест группа)
						time.Sleep(5 * time.Minute) //minute
					}
				}
			}
			time.Sleep(1 * time.Hour) //hour
		}

	}()
	*/

}

func getSettings(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(f)
}

//ПАРСИМ ЛЮДЕЙ У КОТОРЫХ СЕГОДНЯ ДЕНЬ РОЖДЕНИЯ, ОПРЕДЕЛЯЕМ ПОЛ
func getBirthdayJson(list, url string) []Employee {
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
		if strings.HasPrefix(empl.Date, strDate) && strings.HasPrefix(empl.Company, "Е") == false {
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
func getAnonceBirthdayJson(list, url string) []Employee {
	resp, _ := http.Get(fmt.Sprintf("https://tools.aimylogic.com/api/googlesheet2json?sheet=%v&id=%v", list, url))
	defer resp.Body.Close()

	employes := []Employee{}

	err := json.NewDecoder(resp.Body).Decode(&employes)
	if err != nil {
		fmt.Println(err, " body, err")
	}

	employesBirthday := []Employee{} //СТРУКТУРА ЛЮДЕЙ С ДНЁМ РОЖДЕНИЯ
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
		if strings.HasPrefix(empl.Date, strDate) {
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

//ПАРСИМ ЛЮДЕЙ У КОТОРЫЕ МОГУТ БЫТЬ СБОРЩИКАМИ СРЕДСТВ
func getDonationListJson(list, url string) []Employee {
	resp, _ := http.Get(fmt.Sprintf("https://tools.aimylogic.com/api/googlesheet2json?sheet=%v&id=%v", list, url))
	defer resp.Body.Close()

	employes := []Employee{}

	err := json.NewDecoder(resp.Body).Decode(&employes)
	if err != nil {
		fmt.Println(err, " body, err")
	}

	employesBirthday := []Employee{} //СТРУКТУРА ЛЮДЕЙ С ДНЁМ РОЖДЕНИЯ
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

//ПОЛУЧАЕМ ИМЯ В НУЖНОМ ПАДЕЖЕ
func getPrettySuffix(people, padej string) string {
	name := people
	people = strings.Replace(people, " ", "%20", -1)
	resp, err := http.Get(fmt.Sprint("http://ws3.morpher.ru/russian/declension?s=" + people + "&format=json"))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	rodSuffix := rSuffix{}

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

//СЛУЧАЙНОЕ ЧИСЛО ДЛЯ ОПРЕДЕЛЕНИЯ ТЕКСТА СООБЩЕНИЯ
func random(max int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(max)
}

//ПАРСИМ ТАБЛИЦУ С ТЕКСТОМ ПОЗДРАВЛЕНИЙ И РАСПРЕДЕЛЯЕМ ИХ ПО МАССИВАМ
func getCongratArrays(list, url string) ([]TextFirstPart, []TextSecondPart, []TextThirdPart) {
	resp, _ := http.Get(fmt.Sprintf("https://tools.aimylogic.com/api/googlesheet2json?sheet=%v&id=%v", list, url))
	defer resp.Body.Close()

	//МАССИВЫ ДЛЯ ПАРСИНГА
	fTP := []TextFirstPart{}
	sTP := []TextSecondPart{}
	tTP := []TextThirdPart{}

	fTPraw := []TextFirstPart{}
	sTPraw := []TextSecondPart{}
	tTPraw := []TextThirdPart{}

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

//ГЕНЕРИРУЕМ СООБЩЕНИЕ ПО ГУГЛ ТАБЛИЦЕ С ЗАГОТОВКАМИ
func getBirthdayMsg(peoples Employee, list, url string) string {
	//МАССИВЫ СТРУКТУР ЧАСТЕЙ ПОЗДРАВЛЕНИЯ
	fTP, sTP, tTP := getCongratArrays(list, url)

	var text1, text2, text3, text4, text5 string

	//ГЕНЕРИРУЕМ СЛУЧАЙНОЕ ЧИСЛО, И ПО НЕМУ ПОДСТАВЛЯЕМ ЧАСТЬ ТЕКСТА
	text1 = fTP[random(len(fTP))].Congratulation

	//ЕСЛИ В ПЕРВОЙ ЧАСТИ УКАЗАН ПОЛ, ПРОВЕРЯЕМ ПОЛ СОТРУДНИКА
	for strings.HasSuffix(text1, " *Ж") && peoples.Male == "М" {
		text1 = fTP[random(len(fTP))].Congratulation
		time.Sleep(77 * time.Microsecond)
	}
	for strings.HasSuffix(text1, " *М") && peoples.Male == "Ж" {
		text1 = fTP[random(len(fTP))].Congratulation
		time.Sleep(77 * time.Microsecond)
	}
	//ЕСЛИ ПОЛ ОПРЕДЕЛИТЬ НЕ УДАЛОСЬ - НЕ ИСПОЛЬЗУЕМ НАЧАЛЬНЫЕ ФРАЗЫ В КОТОРЫХ ОН УКАЗАН
	for peoples.Male == "?" && (strings.HasSuffix(text1, " *М") || strings.HasSuffix(text1, " *Ж")) {
		text1 = fTP[random(len(fTP))].Congratulation
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
		peoples.Name = getPrettySuffix(peoples.Name, "V")
	}
	if strings.HasSuffix(text1, " *Д") {
		text1 = strings.Replace(text1, " *Д", "", 1)
		peoples.Name = getPrettySuffix(peoples.Name, "D")
	}
	if strings.HasSuffix(text1, " *Р") {
		text1 = strings.Replace(text1, " *Р", "", 1)
		peoples.Name = getPrettySuffix(peoples.Name, "R")
	}

	//ПОЛУЧАЕМ ОТДЕЛ В НУЖНОМ ПАДЕЖЕ
	peoples.Department = getPrettySuffix(peoples.Department, "R")

	text2 = sTP[random(len(sTP))].WishYou
	//ПОЖЕЛАНИЯ ГЕНЕРИРУЕМ ТАК, ЧТОБЫ ОНИ НЕ ПОВТОРЯЛИСЬ
	text3 = tTP[random(len(tTP))].Sentiments
	for text4 == "" || text4 == text3 {
		text4 = tTP[random(len(tTP))].Sentiments
	}
	for text5 == "" || text5 == text4 || text5 == text3 {
		text5 = tTP[random(len(tTP))].Sentiments
	}
	msg := fmt.Sprintf("%v %v из %v! %v %v, %v и %v!", text1, peoples.Name, peoples.Department, text2, text3, text4, text5)
	return msg
}
