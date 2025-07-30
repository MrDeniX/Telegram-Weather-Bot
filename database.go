package main

import (
    "database/sql"
    "log"

    _ "modernc.org/sqlite"
)

type DB = sql.DB

func InitDB() *DB {
    db, err := sql.Open("sqlite", "weather.db")
    if err != nil {
        log.Fatal(err)
    }

    _, err = db.Exec(`CREATE TABLE IF NOT EXISTS users (
        user_id INTEGER PRIMARY KEY,
        city TEXT
    )`)
    if err != nil {
        log.Fatal(err)
    }

    _, err = db.Exec(`CREATE TABLE IF NOT EXISTS subscriptions (
        user_id INTEGER,
        sub_type TEXT,
        custom_hour INTEGER,
        PRIMARY KEY(user_id, sub_type)
    )`)
    if err != nil {
        log.Fatal(err)
    }

    return db
}

func SetUserCity(db *DB, userID int64, city string) {
    _, _ = db.Exec(`
        INSERT INTO users (user_id, city) VALUES (?, ?)
        ON CONFLICT(user_id) DO UPDATE SET city=excluded.city
    `, userID, city)
}

func GetUserCity(db *DB, userID int64) string {
    var city string
    _ = db.QueryRow("SELECT city FROM users WHERE user_id = ?", userID).Scan(&city)
    return city
}

func SetSubscription(db *DB, userID int64, subType string) {
    if subType == "custom" {
        return
    }
    _, _ = db.Exec(`
        INSERT INTO subscriptions (user_id, sub_type) VALUES (?, ?)
        ON CONFLICT(user_id, sub_type) DO NOTHING
    `, userID, subType)
}

func SetCustomHour(db *DB, userID int64, hour int) {
    _, _ = db.Exec(`
        INSERT INTO subscriptions (user_id, sub_type, custom_hour) VALUES (?, 'custom', ?)
        ON CONFLICT(user_id, sub_type) DO UPDATE SET custom_hour=excluded.custom_hour
    `, userID, hour)
}

func UnsetSpecificSubscription(db *DB, userID int64, subType string) {
    _, _ = db.Exec("DELETE FROM subscriptions WHERE user_id = ? AND sub_type = ?", userID, subType)
}

func GetUserSubscriptions(db *DB, userID int64) []string {
    rows, err := db.Query("SELECT sub_type FROM subscriptions WHERE user_id = ?", userID)
    if err != nil {
        return nil
    }
    defer rows.Close()

    var subs []string
    for rows.Next() {
        var sub string
        if err := rows.Scan(&sub); err == nil {
            subs = append(subs, sub)
        }
    }
    return subs
}

func GetCustomHour(db *DB, userID int64) int {
    var hour int
    err := db.QueryRow("SELECT custom_hour FROM subscriptions WHERE user_id = ? AND sub_type = 'custom'", userID).Scan(&hour)
    if err != nil {
        return -1
    }
    return hour
}

func GetSubscribers(db *DB, subType string) []int64 {
    rows, err := db.Query("SELECT user_id FROM subscriptions WHERE sub_type = ?", subType)
    if err != nil {
        return nil
    }
    defer rows.Close()

    var users []int64
    for rows.Next() {
        var id int64
        if err := rows.Scan(&id); err == nil {
            users = append(users, id)
        }
    }
    return users
}

func GetSubscribersByHour(db *DB, hour int) []int64 {
    rows, err := db.Query("SELECT user_id FROM subscriptions WHERE sub_type = 'custom' AND custom_hour = ?", hour)
    if err != nil {
        return nil
    }
    defer rows.Close()

    var users []int64
    for rows.Next() {
        var id int64
        if err := rows.Scan(&id); err == nil {
            users = append(users, id)
        }
    }
    return users
}