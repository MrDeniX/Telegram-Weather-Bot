package main

import (
    "log"
    "os"
    "strconv"
    "time"

    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
    "github.com/joho/godotenv"
)

var bot *tgbotapi.BotAPI

var awaitingCityInput = make(map[int64]bool)
var awaitingCustomTime = make(map[int64]bool)
var menuState = make(map[int64]string)

var channelID string

func main() {
    _ = godotenv.Load()

    channelID = os.Getenv("CHANNEL_ID")
    if channelID == "" {
        log.Panic("CHANNEL_ID не задан в .env")
    }

    var err error
    bot, err = tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_TOKEN"))
    if err != nil {
        log.Panic(err)
    }

    db := InitDB()
    go startScheduler(db)
    go startChannelScheduler()

    u := tgbotapi.NewUpdate(0)
    u.Timeout = 60
    updates := bot.GetUpdatesChan(u)

    for update := range updates {
        if update.Message == nil {
            continue
        }

        chatID := update.Message.Chat.ID
        text := update.Message.Text

        if update.Message.Location != nil {
            lat, lon := update.Message.Location.Latitude, update.Message.Location.Longitude
            forecast, cityName, err := getWeatherByCoordsAndCity(lat, lon)
            if err != nil {
                bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при получении прогноза по геолокации."))
                continue
            }
            SetUserCity(db, chatID, cityName)
            bot.Send(tgbotapi.NewMessage(chatID, "Город сохранён: "+cityName))
            bot.Send(tgbotapi.NewMessage(chatID, forecast))
            continue
        }

        if (menuState[chatID] == "forecast" || menuState[chatID] == "subs" || menuState[chatID] == "citySelection") && text == "🔙 Назад" {
            showMainMenu(chatID)
            continue
        }

        if awaitingCityInput[chatID] {
            SetUserCity(db, chatID, text)
            awaitingCityInput[chatID] = false
            bot.Send(tgbotapi.NewMessage(chatID, "Город сохранён: "+text))
            showMainMenu(chatID)
            continue
        }

        if awaitingCustomTime[chatID] {
            hour, err := strconv.Atoi(text)
            if err != nil || hour < 0 || hour > 23 {
                bot.Send(tgbotapi.NewMessage(chatID, "Введите корректный час от 0 до 23"))
                continue
            }
            SetCustomHour(db, chatID, hour)
            awaitingCustomTime[chatID] = false
            SetSubscription(db, chatID, "custom")
            bot.Send(tgbotapi.NewMessage(chatID, "Подписка включена на "+strconv.Itoa(hour)+":00"))
            showSubscriptionsMenu(chatID)
            continue
        }

        switch menuState[chatID] {
        case "main":
            switch text {
            case "📍 Погода сейчас":
                city := GetUserCity(db, chatID)
                if city == "" {
                    bot.Send(tgbotapi.NewMessage(chatID, "Сначала задайте город!"))
                    continue
                }
                msg, _ := getWeather(city)
                bot.Send(tgbotapi.NewMessage(chatID, msg))

            case "📅 Прогнозы":
                showForecastMenu(chatID)

            case "⏰ Подписки":
                showSubscriptionsMenu(chatID)

            case "🏙 Выбор города":
                showCitySelectionMenu(chatID)

            default:
                bot.Send(tgbotapi.NewMessage(chatID, "Пожалуйста, выберите опцию из меню."))
            }

        case "forecast":
            city := GetUserCity(db, chatID)
            if city == "" {
                bot.Send(tgbotapi.NewMessage(chatID, "Сначала задайте город!"))
                showMainMenu(chatID)
                continue
            }
            switch text {
            case "⏱ Через час":
                msg, _ := getHourlyForecast(city)
                bot.Send(tgbotapi.NewMessage(chatID, msg))
            case "📅 На завтра":
                msg, _ := getTomorrowForecast(city)
                bot.Send(tgbotapi.NewMessage(chatID, msg))
            case "📆 На неделю":
                msg, _ := getWeeklyForecast(city)
                bot.Send(tgbotapi.NewMessage(chatID, msg))
            case "🔙 Назад":
                showMainMenu(chatID)
            default:
                bot.Send(tgbotapi.NewMessage(chatID, "Выберите вариант из меню."))
            }

        case "subs":
            switch text {
            case "📋 Мои подписки":
                showMySubscriptions(db, chatID)

            case "⏰ Утро":
                SetSubscription(db, chatID, "утро")
                bot.Send(tgbotapi.NewMessage(chatID, "Подписка: утро (8:00) включена"))

            case "🌙 Вечер":
                SetSubscription(db, chatID, "вечер")
                bot.Send(tgbotapi.NewMessage(chatID, "Подписка: вечер (20:00) включена"))

            case "🕐 Выбрать время":
                awaitingCustomTime[chatID] = true
                bot.Send(tgbotapi.NewMessage(chatID, "Введите час от 0 до 23 для подписки"))

            case "❌ Отписаться от утра":
                UnsetSpecificSubscription(db, chatID, "утро")
                bot.Send(tgbotapi.NewMessage(chatID, "Подписка на утро отключена"))

            case "❌ Отписаться от вечера":
                UnsetSpecificSubscription(db, chatID, "вечер")
                bot.Send(tgbotapi.NewMessage(chatID, "Подписка на вечер отключена"))

            case "❌ Отписаться от выбранного времени":
                UnsetSpecificSubscription(db, chatID, "custom")
                bot.Send(tgbotapi.NewMessage(chatID, "Кастомная подписка отключена"))

            case "🔙 Назад":
                showMainMenu(chatID)

            default:
                bot.Send(tgbotapi.NewMessage(chatID, "Выберите вариант из меню."))
            }

        case "citySelection":
            switch text {
            case "🏙 Установить город вручную":
                awaitingCityInput[chatID] = true
                bot.Send(tgbotapi.NewMessage(chatID, "Введите город вручную:"))

            case "🔙 Назад":
                showMainMenu(chatID)

            default:
                bot.Send(tgbotapi.NewMessage(chatID, "Выберите вариант из меню или отправьте геолокацию."))
            }

        default:
            showMainMenu(chatID)
        }
    }
}


func showMainMenu(chatID int64) {
    menuState[chatID] = "main"
    msg := tgbotapi.NewMessage(chatID, "Главное меню")
    msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
        tgbotapi.NewKeyboardButtonRow(
            tgbotapi.NewKeyboardButton("📍 Погода сейчас"),
            tgbotapi.NewKeyboardButton("📅 Прогнозы"),
        ),
        tgbotapi.NewKeyboardButtonRow(
            tgbotapi.NewKeyboardButton("⏰ Подписки"),
            tgbotapi.NewKeyboardButton("🏙 Выбор города"),
        ),
    )
    bot.Send(msg)
}

func showForecastMenu(chatID int64) {
    menuState[chatID] = "forecast"
    msg := tgbotapi.NewMessage(chatID, "Выберите прогноз")
    msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
        tgbotapi.NewKeyboardButtonRow(
            tgbotapi.NewKeyboardButton("⏱ Через час"),
            tgbotapi.NewKeyboardButton("📅 На завтра"),
        ),
        tgbotapi.NewKeyboardButtonRow(
            tgbotapi.NewKeyboardButton("📆 На неделю"),
        ),
        tgbotapi.NewKeyboardButtonRow(
            tgbotapi.NewKeyboardButton("🔙 Назад"),
        ),
    )
    bot.Send(msg)
}

func showSubscriptionsMenu(chatID int64) {
    menuState[chatID] = "subs"
    msg := tgbotapi.NewMessage(chatID, "Меню подписок")
    msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
        tgbotapi.NewKeyboardButtonRow(
            tgbotapi.NewKeyboardButton("📋 Мои подписки"),
        ),
        tgbotapi.NewKeyboardButtonRow(
            tgbotapi.NewKeyboardButton("⏰ Утро"),
            tgbotapi.NewKeyboardButton("🌙 Вечер"),
        ),
        tgbotapi.NewKeyboardButtonRow(
            tgbotapi.NewKeyboardButton("🕐 Выбрать время"),
        ),
        tgbotapi.NewKeyboardButtonRow(
            tgbotapi.NewKeyboardButton("🔙 Назад"),
        ),
    )
    bot.Send(msg)
}

func showCitySelectionMenu(chatID int64) {
    menuState[chatID] = "citySelection"
    msg := tgbotapi.NewMessage(chatID, "Выбор города")
    msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
        tgbotapi.NewKeyboardButtonRow(
            tgbotapi.NewKeyboardButton("🏙 Установить город вручную"),
        ),
        tgbotapi.NewKeyboardButtonRow(
            tgbotapi.NewKeyboardButtonLocation("📡 Отправить геолокацию"),
        ),
        tgbotapi.NewKeyboardButtonRow(
            tgbotapi.NewKeyboardButton("🔙 Назад"),
        ),
    )
    bot.Send(msg)
}

func showMySubscriptions(db *DB, chatID int64) {
    subs := GetUserSubscriptions(db, chatID)
    if len(subs) == 0 {
        bot.Send(tgbotapi.NewMessage(chatID, "У тебя нет активных подписок."))
        return
    }

    text := "Твои подписки:\n"
    rows := [][]tgbotapi.KeyboardButton{}

    for _, sub := range subs {
        switch sub {
        case "утро":
            text += "✅ Утро (8:00)\n"
            rows = append(rows, tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("❌ Отписаться от утра")))
        case "вечер":
            text += "✅ Вечер (20:00)\n"
            rows = append(rows, tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("❌ Отписаться от вечера")))
        case "custom":
            hour := GetCustomHour(db, chatID)
            text += "✅ Выбранное время: " + strconv.Itoa(hour) + ":00\n"
            rows = append(rows, tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("❌ Отписаться от выбранного времени")))
        }
    }

    rows = append(rows, tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("🔙 Назад")))

    msg := tgbotapi.NewMessage(chatID, text)
    msg.ReplyMarkup = tgbotapi.ReplyKeyboardMarkup{
        Keyboard:       rows,
        ResizeKeyboard: true,
    }
    bot.Send(msg)
}


func startScheduler(db *DB) {
    go func() {
        for {
            now := time.Now()
            hour := now.Hour()

            if hour == 8 {
                subs := GetSubscribers(db, "утро")
                sendWeatherToUsers(db, subs)
            }

            if hour == 20 {
                subs := GetSubscribers(db, "вечер")
                sendWeatherToUsers(db, subs)
            }

            subs := GetSubscribersByHour(db, hour)
            sendWeatherToUsers(db, subs)

            time.Sleep(time.Minute)
        }
    }()
}

func sendWeatherToUsers(db *DB, users []int64) {
    for _, userID := range users {
        city := GetUserCity(db, userID)
        if city == "" {
            continue
        }
        forecast, err := getWeather(city)
        if err != nil {
            continue
        }
        msg := tgbotapi.NewMessage(userID, "Прогноз погоды:\n"+forecast)
        bot.Send(msg)
    }
}


func startChannelScheduler() {
    go func() {
        for {
            now := time.Now()
            hour, min := now.Hour(), now.Minute()

            if min == 0 {
                sendHourlyWeatherToChannel()
            }

            if hour == 7 && min == 0 {
                sendDailyForecastToChannel()
            }

            time.Sleep(time.Second * 30)
        }
    }()
}

func sendDailyForecastToChannel() {
    city := "Симферополь"
    msg, err := getWeeklyForecast(city)
    if err != nil {
        log.Println("Ошибка получения недельного прогноза для канала:", err)
        return
    }
    fullMsg := "🌤 Прогноз погоды на неделю для " + city + ":\n" + msg
    m := tgbotapi.NewMessageToChannel(channelID, fullMsg)
    if _, err := bot.Send(m); err != nil {
        log.Println("Ошибка отправки недельного прогноза в канал:", err)
    }
}

func sendHourlyWeatherToChannel() {
    city := "Симферополь"
    msg, err := getWeather(city)
    if err != nil {
        log.Println("Ошибка получения текущей погоды для канала:", err)
        return
    }
    fullMsg := "⏰ Текущая погода в " + city + ":\n" + msg
    m := tgbotapi.NewMessageToChannel(channelID, fullMsg)
    if _, err := bot.Send(m); err != nil {
        log.Println("Ошибка отправки текущей погоды в канал:", err)
    }
}
