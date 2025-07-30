# Telegram-Weather-Bot

A simple and user-friendly Telegram bot written in Go that provides current weather and forecasts for different time periods.

---

## 🚀 Features

- Get current weather by city  
- Weather forecast for the next hour, tomorrow, and the week ahead  
- Easy-to-use menu for selecting options  
- Automatic weather forecast posting to a channel (e.g., for Simferopol city)  

---

## 🛠 Technologies

- Programming language: **Go**  
- Telegram Bot API  
- External weather API integration  
- Local data storage (e.g., SQLite)  

---

## 🔐 Configuration Files

### `.env`

This file stores your sensitive configuration data such as API keys and tokens.  
**Important:** Do **not** commit this file to your repository.

Create a `.env` file in the root directory of the project with the following variables:

```env
TELEGRAM_TOKEN=your_telegram_bot_token_here
WEATHER_API_KEY=your_weather_api_key_here
CHANNEL_ID=your_channel_id