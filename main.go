package main

import (
	"TelegramBot-congratulator/entity"
	"TelegramBot-congratulator/helpers"
	"database/sql"
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"strconv"
	"strings"
	"time"
)

var numericKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Описание"),
		tgbotapi.NewKeyboardButton("Статус"),
		tgbotapi.NewKeyboardButton("Подписчики"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("СписокДР"),
		tgbotapi.NewKeyboardButton("Удалиться"),
		tgbotapi.NewKeyboardButton("Фильтры"),
	),
)

func main() {

	var BotSets entity.BotSettings

	//СОЗДАНИЕ БД
	database, _ := sql.Open("sqlite3", "./TelegramBot_Congratulator.db")
	statement, _ := database.Prepare("CREATE TABLE IF NOT EXISTS people (id INTEGER PRIMARY KEY, chat_id INTEGER,username TEXT, department_filter TEXT)")
	statement.Exec()

	//ИЗВЛЕКАЕМ ИЗ ФАЙЛА С НАСТРОЙКАМИ ПОЛЯ
	bs, err := helpers.GetSettings("settings.json")
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

	bot.Debug = false

	log.Printf("Authorized on account %s", bot.Self.UserName)

	//СПАМ В ЛИЧКУ ПОДПИСАВШИМСЯ
	go func() {
		for {
			//АНОНС ТОЛЬКО В ПЕРИОД 11-12
			currentTime := time.Now()
			if currentTime.Hour() == 11 {

				//ПОЛУЧАЕМ СПИСОК ЛЮДЕЙ У КОГО ДР ЗАВТРА
				birthdayTomorrow := helpers.GetAnonceBirthdayJson(BotSets.Google_sheet_bday_list, BotSets.Google_sheet_bday_url)

				//ЕСЛИ ЛЮДЕЙ У КОТОРЫЙ ДР ЗАВТРА ХОТЯБЫ 1 ДЕЛАЕМ РАССЫЛКУ
				if len(birthdayTomorrow) > 0 {
					//ПОЛУЧАЕМ СПИСОК ЛЮДЕЙ КОТОРЫЕ ГОТОВЫ СОБИРАТЬ ДЕНЬГИ
					donatorList := helpers.GetDonationListJson(BotSets.Google_sheet_bday_list, BotSets.Google_sheet_bday_url)

					//ПОЛУЧАЕМ СПИСОК ЛЮДЕЙ КОТОРЫМ ДЕЛАТЬ РАССЫЛКУ
					u := []entity.UsersForSpam{}
					a := entity.UsersForSpam{}
					rows, _ := database.Query("SELECT chat_id, username FROM people")

					for rows.Next() {
						rows.Scan(&a.ChatID, &a.Name)
						u = append(u, a)
					}
					rows.Close()

					//ИТЕРИРУЕМСЯ ПО ЛЮДЯМ У КОТОРЫХ ДР ЗАВТРА
					for _, peoples := range birthdayTomorrow {

						//СКЛОНЯЕМ ИМЯ И ОТДЕЛ
						nameR := helpers.GetPrettySuffix(peoples.Name, "R")
						departmentR := helpers.GetPrettySuffix(peoples.Department, "R")

						var myDonator entity.Employee

						if strings.Contains(peoples.Donater, "@") {
							for _, donator := range donatorList {
								if donator.Mail == peoples.Donater && donator.Name != peoples.Name {
									myDonator = donator
									break
								}
							}
						}
						//ИЩЕМ ЧЕЛОВЕКА ОТВЕТСТВЕННОГО ЗА СБОР СРЕДСТВ ВНУТРИ ОТДЕЛА
						if myDonator.Name == "" {
							for _, donator := range donatorList {
								if donator.Department == peoples.Department && donator.Name != peoples.Name {
									myDonator = donator
									break
								}
							}
						}
						//ЕСЛИ НЕ НАШЛИ КОМУ ПЕРЕВОДИТЬ ИЗ ОТДЕЛА ИМЕНИННИКА, ИЩЕМ В ОТДЕЛЕ HR
						if myDonator.Name == "" {
							for _, donator := range donatorList {
								if donator.Department == "Отдел по работе с персоналом" && donator.Name != peoples.Name {
									myDonator = donator
									break
								}
							}
						}

						//РАССЫЛАЕМ ПОДПИСЧИКАМ ИЗ БД
						for _, follower := range u {
							if peoples.Name != follower.Name {
								if departmentR != "" {
									if myDonator.Name != "" { //ЕСЛИ НАШЛИ ЧЕЛОВЕКА, КОТОРЫЙ ГОТОВ СОБИРАТЬ ДЕНЬГИ
										msg := fmt.Sprintf("Завтра день рождения у %s из %s!\nПодарок собирает %s.\nПринимает переводы по номеру %v\nЕсли захочешь поучаствовать - Укажи комментарий, для кого подарок :)", nameR, departmentR, myDonator.Name, myDonator.Telephone)
										bot.Send(tgbotapi.NewMessage(follower.ChatID, msg))
									} else { //ЕСЛИ НЕ НАШЛИ ЧЕЛОВЕКА, КОТОРЫЙ ГОТОВ СОБИРАТЬ ДЕНЬГИ
										msg := fmt.Sprintf("Завтра день рождения у %s из %s!\nЯ не нашёл никого, кто будет собирать подарки, поэтому просто поздравим словестно!)", nameR, departmentR)
										bot.Send(tgbotapi.NewMessage(follower.ChatID, msg))
									}
								} else {
									msg := fmt.Sprintf("Завтра день рождения у %s!\nПодарок собирает %s.\nПринимает переводы по номеру %v\nЕсли захочешь поучаствовать - Укажи комментарий, для кого подарок :)", nameR, myDonator.Name, myDonator.Telephone)

									//https://web3.online.sberbank.ru/transfers/client ССЫЛКА НА ЛК СБЕРА
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
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Не понял Вас, Жду сообщения вида Регистрация Иван Иванов"))
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
					text := fmt.Sprintf("Регистрация завершена, %v", userInputName)

					msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
					msg.ReplyMarkup = numericKeyboard
					bot.Send(msg)

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

		case "/UNSUBSCRIBE", "УДАЛИТЬСЯ": //УДАЛИТЬСЯ ИЗ БД

			//СЧИТЫВАНИЕ ИЗ БАЗЫ
			data1, _ := database.Query("SELECT chat_id FROM people WHERE chat_id = ?", update.Message.Chat.ID)
			var chatId int

			data1.Next()
			data1.Scan(&chatId)
			data1.Close()
			if chatId == 0 {
				//ЕСЛИ СТРОКИ ВЫВОДИМ СООБЩЕНИЕ В ЧАТ

				text := fmt.Sprintf("Вас не было в рассылке")

				msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
				msg.ReplyMarkup = numericKeyboard
				bot.Send(msg)

			} else {
				//ЕСЛИ СТРОКА ЕСТЬ - УДАЛЯЕМ ПОЛЬЗОВАТЕЛЯ
				_, err := database.Exec("DELETE FROM people WHERE chat_id = ?", update.Message.Chat.ID)
				if err != nil {
					fmt.Println("Ошибка удаления")
				}
				//ВЫВОД В ЧАТ
				text := "Вы удалены из рассылки :("
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
				msg.ReplyMarkup = numericKeyboard
				bot.Send(msg)
			}

		case "/STATUS", "СТАТУС":

			//СЧИТЫВАНИЕ ИЗ БАЗЫ
			data1, _ := database.Query("SELECT chat_id, username FROM people WHERE chat_id = ?", update.Message.Chat.ID)
			var chatId int
			var username string

			data1.Next()
			data1.Scan(&chatId, &username)
			data1.Close()
			if username != "" {
				text := fmt.Sprintf("Ваше имя в рассылке %v", username)

				msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
				msg.ReplyMarkup = numericKeyboard
				bot.Send(msg)
			} else {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Вас нет в рассылке :( для регистрации введите\nРегистрация Имя Фамилия"))
			}

		case "/DESCRIPTION", "ОПИСАНИЕ":
			text := fmt.Sprintf("Описание комманд:\nРегистация Имя Фамилия - зарегистрироваться или обновить данные" +
				"\nУдалиться - удалиться из рассылок\nСтатус - ваше имя в рассылке\nПодписчики - список подписавшихся" +
				"\nСписокдр август - покажу у кого ДР в этом месяце. Если месяц не указан - покажу текущий месяц")

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
			msg.ReplyMarkup = numericKeyboard
			bot.Send(msg)

		case "/FILTERS", "ФИЛЬТРЫ":
			msg := fmt.Sprintf("На данный момент фильтры недоступны, придётся смотреть на все дни рождения :)")
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))

		case "/START":
			text := fmt.Sprintf("Привет! Я помогу тебе поздравлять твоих коллег без десятков надоедливых чатов :)\n" +
				"Для начала, зарегистрируйся. Примерно так: \nРегистрация Иван Иванов (сначала имя, потом фамилия)\n" +
				"Чтобы узнать что я умею введи Описание\n" +
				"Хорошего тебе дня!")

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
			msg.ReplyMarkup = numericKeyboard
			bot.Send(msg)

		case "ОБЪЯВЛЕНИЕВСЕМ":
			if len(command) > 1 {
				if command[1] == BotSets.Anonce_pass {
					//trim := fmt.Sprintf("%v %v ", command[0], BotSets.Anonce_pass)
					//msg := strings.TrimPrefix(update.Message.Text, trim)
					msg := update.Message.Text[len(command[0])+len(command[1])+2:]

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
					trim := fmt.Sprintf("%v %v %v ", command[0], BotSets.Anonce_pass, ignoreName)
					text := strings.TrimPrefix(update.Message.Text, trim)

					rows, _ := database.Query("SELECT chat_id, username FROM people")
					var chatID int64
					var userName string

					//ПОЛУЧАЕМ СПИСОК ЧАТОВ ПОДПИСЧИКОВ И РАССЫЛАЕМ СООБЩЕНИЕ
					for rows.Next() {
						rows.Scan(&chatID, &userName)
						if userName != ignoreName {
							bot.Send(tgbotapi.NewMessage(chatID, text))
							msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
							msg.ReplyMarkup = numericKeyboard
							bot.Send(msg)
						}
					}

				} else {
					//ЕСЛИ ПАРОЛЬ НЕВЕРНЫЙ ТО
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Неверный пароль, попробуйте снова"))
				}
			} else {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Неверная команда, Введите ОБЪЯВЛЕНИЕКРОМЕ (Пароль) Имя Текст"))
			}

		case "/SUBSCRIBERS", "ПОДПИСЧИКИ": //ВЫВОДИТ СПИСОК ВСЕХ ПОДПИСАВШИХСЯ
			text := ""
			var sum int
			//ЗАПРАШИВАЕМ ИЗ БД ВСЕ ИМЕНА
			rows, _ := database.Query("SELECT username FROM people")
			var followers string

			//ПРОВЕРЯЕМ ВСЕ ДАННЫЕ В БАЗЕ ИМЁН
			for rows.Next() {
				rows.Scan(&followers)

				sum += 1
				if text != "" {
					text += fmt.Sprintf("\n%s", followers)
				} else {
					text += fmt.Sprintf("%s", followers)
				}
			}
			text += fmt.Sprintf("\nЧисло подписчиков: %v", sum)
			//ВЫВОД В ЧАТ
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
			msg.ReplyMarkup = numericKeyboard
			bot.Send(msg)
		case "OPEN":
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Открыто")
			msg.ReplyMarkup = numericKeyboard
			bot.Send(msg)
		case "CLOSE":
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
			msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
			bot.Send(msg)

		case "ОТДЕЛЫ":
			text := helpers.Departments(BotSets.Google_sheet_bday_list, BotSets.Google_sheet_bday_url)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
			msg.ReplyMarkup = numericKeyboard
			bot.Send(msg)

		case "/BIRTHDAYLIST", "СПИСОКДР":
			if len(command) < 3 { //ЕСЛИ ПОЛЬЗОВАТЕЛЬ ВВЕЛ КОМАНДУ ВЕРНО

				var month, text string

				if len(command) == 1 { //ЕСЛИ ПОЛЬЗОВАТЕЛЬ НЕ ВВЕЛ МЕСЯЦ
					//КОНВЕРТИРУЕМ ТЕКУЩИЙ МЕСЯЦ В СТРИНГ И ПЕРЕДАЕМ В ФУНКЦИЮ
					month = strconv.Itoa(int(time.Now().Month()))
					switch int((time.Now().Month())) {
					case 1:
						text += "Январь"
					case 2:
						text += "Февраль"
					case 3:
						text += "Март"
					case 4:
						text += "Апрель"
					case 5:
						text += "Май"
					case 6:
						text += "Июнь"
					case 7:
						text += "Июль"
					case 8:
						text += "Август"
					case 9:
						text += "Сентябрь"
					case 10:
						text += "Октябрь"
					case 11:
						text += "Ноябрь"
					case 12:
						text += "Декабрь"
					}
				} else { //ЕСЛИ УКАЗАЛ МЕСЯЦ - ПЕРЕДАЕМ В ФУНКЦИЮ ВВОД, А СООБЩЕНИЕ НАЧИНАЕМ С ЕГО ВВОДА
					month = command[1]
					text += command[1]
				}

				//ПОЛУЧАЕМ МАССИВ ЛЮДЕЙ С ДР В УКАЗАННОМ МЕСЯЦЕ
				birthdayList := helpers.GetBirthdayMonthListJson(BotSets.Google_sheet_bday_list, BotSets.Google_sheet_bday_url, month)

				//ОБРАБАТЫВАЕМ ПОЛУЧЕННЫЙ СПИСОК ЛЮДЕЙ
				if len(birthdayList) < 1 {
					//ЕСЛИ В СПИСКЕ НИКОГО - ВЫДАЕМ ОШИБКУ
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Ничего не нашел, проверьте месяц"))
				} else {
					//ЕСЛИ БОЛЬШЕ ОДНОГО ЧЕЛОВЕКА В СПИСКЕ - СОРТИРУЕМ СПИСОК ПО ДАТЕ
					if len(birthdayList) > 1 {
						for i := 0; i < len(birthdayList)-1; i++ {
							for j := i + 1; j < len(birthdayList); j++ {
								date1arr := strings.Split(birthdayList[i].Date, " ")
								date2arr := strings.Split(birthdayList[j].Date, " ")
								date1, err := strconv.Atoi(date1arr[0])
								if err != nil {
									fmt.Println(err)
								}
								date2, err := strconv.Atoi(date2arr[0])
								if err != nil {
									fmt.Println(err)
								}
								if date2 < date1 {
									birthdayList[i], birthdayList[j] = birthdayList[j], birthdayList[i]
								}
							}
						}
					}

					//ИТЕРИРУЕМСЯ ПО ЛЮДЯМ У КОТОРЫХ ДР В ТЕКУЩЕМ МЕСЯЦЕ
					for _, peoples := range birthdayList {
						text += fmt.Sprintf("\n%v - %v - %v", peoples.Date, peoples.Name, peoples.Department)
					}

					//ВЫВОД В ЧАТ СООБЩЕНИЕ
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
					msg.ReplyMarkup = numericKeyboard
					bot.Send(msg)
				}
			} else { //ЕСЛИ ВВЕДЕНО БОЛЬШЕ СЛОВ ЧЕМ НУЖНО
				text := "Не понял вас :( Введите команду вида - Списокдр Январь"
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
				msg.ReplyMarkup = numericKeyboard
				bot.Send(msg)
			}

		default:
			text := "Команда не найдена, посмотрите Описание"
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
			msg.ReplyMarkup = numericKeyboard
			bot.Send(msg)
		}
	}
	//КОМАНДЫ КОНЧИЛАСЬ

	//СПАМ В ОБЩИЙ ЧАТ
	//TODO: Вынести в настройки бота отключение спама по флагу
	//TODO: Добавить анонс пятничных игр
	/*
		go func() {
			for {
				//ПОЗДРАВЛЕНИЕ  ТОЛЬКО В ПЕРИОД 10-11
				currentTime := time.Now()

				if currentTime.Hour() == 9 {

					birthdayToday := helpers.GetBirthdayJson(BotSets.Google_sheet_bday_list, BotSets.Google_sheet_bday_url)

					if len(birthdayToday) > 0 {

						for _, peoples := range birthdayToday {
							fmt.Println(peoples)
							msg := helpers.GetBirthdayMsg(peoples, BotSets.Google_sheet_text_list, BotSets.Google_sheet_text_url)
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
