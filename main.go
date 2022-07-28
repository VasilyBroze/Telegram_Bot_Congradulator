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
	"strings"
	"time"
)

func main() {

	var BotSets entity.BotSettings

	//СОЗДАНИЕ БД
	database, _ := sql.Open("sqlite3", "./TelegramBot_Congratulator.db")
	statement, _ := database.Prepare("CREATE TABLE IF NOT EXISTS people (id INTEGER PRIMARY KEY, chat_id INTEGER,username TEXT)")
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
		u := []entity.UsersForSpam{}
		a := entity.UsersForSpam{}
		rows, _ := database.Query("SELECT chat_id, username FROM people")

		//ПРОВЕРЯЕМ ВСЕ ДАННЫЕ В ТАБЛИЦЕ ПО ЧАТ ID
		for rows.Next() {
			rows.Scan(&a.ChatID, &a.Name)
			u = append(u, a)
		}
		for {
			//АНОНС ТОЛЬКО В ПЕРИОД 10-11
			currentTime := time.Now()

			if currentTime.Hour() == 11 {
				//ПОЛУЧАЕМ СПИСОК ЛЮДЕЙ У КОГО ДР ЗАВТРА
				birthdayTomorrow := helpers.GetAnonceBirthdayJson(BotSets.Google_sheet_bday_list, BotSets.Google_sheet_bday_url)
				//ЕСЛИ ЛЮДЕЙ У КОТОРЫЙ ДР ЗАВТРА ХОТЯБЫ 1 ДЕЛАЕМ РАССЫЛКУ
				if len(birthdayTomorrow) > 0 {
					//ПОЛУЧАЕМ СПИСОК ЛЮДЕЙ КОТОРЫЕ ГОТОВЫ СОБИРАТЬ ДЕНЬГИ
					donatorList := helpers.GetDonationListJson(BotSets.Google_sheet_bday_list, BotSets.Google_sheet_bday_url)
					//ИТЕРИРУЕМСЯ ПО ЛЮДЯМ У КОТОРЫХ ДР
					for _, peoples := range birthdayTomorrow {

						//СКЛОНЯЕМ ИМЯ И ОТДЕЛ
						nameR := helpers.GetPrettySuffix(peoples.Name, "R")
						departmentR := helpers.GetPrettySuffix(peoples.Department, "R")
						//ИЩЕМ ЧЕЛОВЕКА ОТВЕТСТВЕННОГО ЗА СБОР СРЕДСТВ ВНУТРИ ОТДЕЛА
						var myDonator entity.Employee
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
									msg := fmt.Sprintf("Завтра день рождения у %s из %s!\nПодарок собирает %s.\nПринимает переводы по номеру %v\nЕсли захочешь поучаствовать - Укажи комментарий, для кого подарок :)\nhttps://web3.online.sberbank.ru/transfers/client", nameR, departmentR, myDonator.Name, myDonator.Telephone)
									bot.Send(tgbotapi.NewMessage(follower.ChatID, msg))
								} else {
									msg := fmt.Sprintf("Завтра день рождения у %s!\nПодарок собирает %s.\nПринимает переводы по номеру %v\nЕсли захочешь поучаствовать - Укажи комментарий, для кого подарок :)\nhttps://web3.online.sberbank.ru/transfers/client", nameR, myDonator.Name, myDonator.Telephone)
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
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Вы удалены из рассылки :("))
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
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Вас нет в рассылке :( для регистрации введите\nРегистрация Имя Фамилия"))
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
					//ЕСЛИ ПАРОЛЬ НЕВЕРНЫЙ ТО
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Неверный пароль, попробуйте снова"))
				}
			} else {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Неверная команда, Введите ОБЪЯВЛЕНИЕКРОМЕ (Пароль) Имя Текст"))
			}

		case "ПОДПИСЧИКИ": //ВЫВОДИТ СПИСОК ВСЕХ ПОДПИСАВШИХСЯ
			msg := ""
			var sum int
			//ЗАПРАШИВАЕМ ИЗ БД ВСЕ ИМЕНА
			rows, _ := database.Query("SELECT username FROM people")
			var followers string

			//ПРОВЕРЯЕМ ВСЕ ДАННЫЕ В БАЗЕ ИМЁН
			for rows.Next() {
				rows.Scan(&followers)

				sum += 1
				if msg != "" {
					msg += fmt.Sprintf("\n%s", followers)
				} else {
					msg += fmt.Sprintf("%s", followers)
				}
			}
			msg += fmt.Sprintf("\nЧисло подписчиков: %v", sum)
			//ВЫВОД В ЧАТ
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))

		default:
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Команда не найдена, посмотрите Описание"))
		}
	}
	//КОМАНДЫ КОНЧИЛАСЬ

	//СПАМ В ОБЩИЙ ЧАТ
	//TODO: Вынести в настройки бота отключение спама по флагу
	/*
		go func() {
			for {
				//ПОЗДРАВЛЕНИЕ ТОЛЬКО В ПЕРИОД 10-11
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
