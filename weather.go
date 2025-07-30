package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"time"
)

func getWeather(city string) (string, error) {
	key := os.Getenv("OPENWEATHER_TOKEN")
	url := fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?q=%s&appid=%s&units=metric&lang=ru", city, key)

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var data struct {
		Main struct {
			Temp float64 `json:"temp"`
		} `json:"main"`
		Weather []struct {
			Description string `json:"description"`
		} `json:"weather"`
		Name string `json:"name"`
	}

	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return "", err
	}

	if len(data.Weather) == 0 {
		return "", fmt.Errorf("нет данных о погоде")
	}

	return fmt.Sprintf("🌤 В %s сейчас %.1f°C, %s", data.Name, data.Main.Temp, data.Weather[0].Description), nil
}

func getWeatherByCoordsAndCity(lat, lon float64) (string, string, error) {
	key := os.Getenv("OPENWEATHER_TOKEN")
	url := fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?lat=%f&lon=%f&appid=%s&units=metric&lang=ru", lat, lon, key)

	resp, err := http.Get(url)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	var data struct {
		Main struct {
			Temp float64 `json:"temp"`
		} `json:"main"`
		Weather []struct {
			Description string `json:"description"`
		} `json:"weather"`
		Name string `json:"name"`
	}

	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return "", "", err
	}

	if len(data.Weather) == 0 {
		return "", "", fmt.Errorf("нет данных о погоде")
	}

	forecast := fmt.Sprintf("🌤 В %s сейчас %.1f°C, %s", data.Name, data.Main.Temp, data.Weather[0].Description)
	return forecast, data.Name, nil
}

func getHourlyForecast(city string) (string, error) {
	key := os.Getenv("OPENWEATHER_TOKEN")
	url := fmt.Sprintf("https://api.openweathermap.org/data/2.5/forecast?q=%s&appid=%s&units=metric&lang=ru", city, key)

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var data struct {
		List []struct {
			DtTxt  string `json:"dt_txt"`
			Main   struct {
				Temp float64 `json:"temp"`
			} `json:"main"`
			Weather []struct {
				Description string `json:"description"`
			} `json:"weather"`
		} `json:"list"`
	}

	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return "", err
	}

	if len(data.List) < 1 {
		return "", fmt.Errorf("нет данных прогноза")
	}

	entry := data.List[0]

	return fmt.Sprintf("⏰ Прогноз через час: %.1f°C, %s", entry.Main.Temp, entry.Weather[0].Description), nil
}

func getTomorrowForecast(city string) (string, error) {
	key := os.Getenv("OPENWEATHER_TOKEN")
	url := fmt.Sprintf("https://api.openweathermap.org/data/2.5/forecast?q=%s&appid=%s&units=metric&lang=ru", city, key)

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var data struct {
		List []struct {
			DtTxt  string `json:"dt_txt"`
			Main   struct {
				Temp float64 `json:"temp"`
			} `json:"main"`
			Weather []struct {
				Description string `json:"description"`
			} `json:"weather"`
		} `json:"list"`
	}

	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return "", err
	}

	loc := time.Now().Location()
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	var temps []float64
	var descriptions []string

	for _, entry := range data.List {
		if len(entry.DtTxt) < 10 {
			continue
		}
		date := entry.DtTxt[:10]
		if date == tomorrow {
			temps = append(temps, entry.Main.Temp)
			if len(entry.Weather) > 0 {
				descriptions = append(descriptions, entry.Weather[0].Description)
			}
		}
	}

	if len(temps) == 0 {
		return "", fmt.Errorf("нет данных для прогноза на завтра")
	}

	minTemp, maxTemp := temps[0], temps[0]
	for _, t := range temps {
		if t < minTemp {
			minTemp = t
		}
		if t > maxTemp {
			maxTemp = t
		}
	}

	desc := mostFrequent(descriptions)

	dt, err := time.ParseInLocation("2006-01-02", tomorrow, loc)
	if err != nil {
		dt = time.Now().AddDate(0, 0, 1)
	}

	weekdayStr := weekdayRu(dt.Weekday())

	return fmt.Sprintf("📅 Завтра (%s): %d~%d°C, %s", weekdayStr, int(minTemp), int(maxTemp), desc), nil
}

func getWeeklyForecast(city string) (string, error) {
	key := os.Getenv("OPENWEATHER_TOKEN")
	url := fmt.Sprintf("https://api.openweathermap.org/data/2.5/forecast?q=%s&appid=%s&units=metric&lang=ru", city, key)

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var data struct {
		List []struct {
			DtTxt  string `json:"dt_txt"`
			Main   struct {
				Temp float64 `json:"temp"`
			} `json:"main"`
			Weather []struct {
				Description string `json:"description"`
			} `json:"weather"`
		} `json:"list"`
	}

	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return "", err
	}

	type daySummary struct {
		date         string
		temps        []float64
		descriptions []string
	}

	dayMap := make(map[string]*daySummary)

	for _, entry := range data.List {
		dt, err := time.Parse("2006-01-02 15:04:05", entry.DtTxt)
		if err != nil {
			continue
		}
		dayKey := dt.Format("2006-01-02")

		ds, exists := dayMap[dayKey]
		if !exists {
			ds = &daySummary{date: dayKey}
			dayMap[dayKey] = ds
		}
		ds.temps = append(ds.temps, entry.Main.Temp)
		if len(entry.Weather) > 0 {
			ds.descriptions = append(ds.descriptions, entry.Weather[0].Description)
		}
	}

	keys := make([]string, 0, len(dayMap))
	for k := range dayMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := "Прогноз на 7 дней:\n"
	loc := time.Now().Location()

	for _, day := range keys {
		ds := dayMap[day]
		minTemp, maxTemp := ds.temps[0], ds.temps[0]
		for _, t := range ds.temps {
			if t < minTemp {
				minTemp = t
			}
			if t > maxTemp {
				maxTemp = t
			}
		}
		desc := mostFrequent(ds.descriptions)

		dt, err := time.ParseInLocation("2006-01-02", ds.date, loc)
		if err != nil {
			dt = time.Now()
		}

		formattedDate := fmt.Sprintf("%d %s, %s", dt.Day(), monthRu(dt.Month()), weekdayRu(dt.Weekday()))

		result += fmt.Sprintf("📅 %s: %d~%d°C, %s\n", formattedDate, int(minTemp), int(maxTemp), desc)
	}

	return result, nil
}

func mostFrequent(arr []string) string {
	count := make(map[string]int)
	maxCount := 0
	var mostCommon string
	for _, v := range arr {
		count[v]++
		if count[v] > maxCount {
			maxCount = count[v]
			mostCommon = v
		}
	}
	return mostCommon
}

func monthRu(m time.Month) string {
	months := []string{
		"января", "февраля", "марта", "апреля", "мая", "июня",
		"июля", "августа", "сентября", "октября", "ноября", "декабря",
	}
	return months[m-1]
}

func weekdayRu(w time.Weekday) string {
	days := []string{
		"воскресенье", "понедельник", "вторник", "среда", "четверг", "пятница", "суббота",
	}
	return days[w]
}
