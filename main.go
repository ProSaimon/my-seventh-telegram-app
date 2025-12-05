package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	LastSeen  int64  `json:"last_seen"`
}

var (
	users     = make(map[int64]*User)
	usersLock sync.RWMutex
)

func homePage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "telegram.html") // Используем telegram.html
}

func getUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	usersLock.RLock()
	defer usersLock.RUnlock()
	
	now := time.Now().Unix()
	activeUsers := []*User{}
	
	for _, user := range users {
		if now-user.LastSeen <= 10 {
			activeUsers = append(activeUsers, user)
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(activeUsers)
}

func updateUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "POST" {
		http.Error(w, "Только POST", http.StatusMethodNotAllowed)
		return
	}
	
	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Ошибка JSON", http.StatusBadRequest)
		return
	}
	
	var userID int64
	switch id := data["id"].(type) {
	case float64:
		userID = int64(id)
	case string:
		if num, err := strconv.ParseInt(id, 10, 64); err == nil {
			userID = num
		} else {
			userID = int64(hashString(id))
		}
	default:
		http.Error(w, "Неверный ID", http.StatusBadRequest)
		return
	}
	
	user := &User{
		ID:        userID,
		FirstName: getString(data, "first_name", "Пользователь"),
		LastName:  getString(data, "last_name", ""),
		Username:  getString(data, "username", ""),
		LastSeen:  time.Now().Unix(),
	}
	
	usersLock.Lock()
	users[userID] = user
	usersLock.Unlock()
	
	log.Printf("👤 Пользователь: %d (%s)", userID, user.FirstName)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"id":     userID,
	})
}

func hashString(s string) uint32 {
	var hash uint32 = 5381
	for i := 0; i < len(s); i++ {
		hash = ((hash << 5) + hash) + uint32(s[i])
	}
	return hash
}

func getString(data map[string]interface{}, key, defaultValue string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

func cleanup() {
	for {
		time.Sleep(10 * time.Second)
		
		usersLock.Lock()
		now := time.Now().Unix()
		removed := 0
		
		for id, user := range users {
			if now-user.LastSeen > 30 {
				delete(users, id)
				removed++
			}
		}
		
		currentUsers := len(users)
		usersLock.Unlock()
		
		if removed > 0 {
			log.Printf("🧹 Удалено %d неактивных, осталось %d", removed, currentUsers)
		}
	}
}

func main() {
	go cleanup()
	
	http.HandleFunc("/", homePage)
	http.HandleFunc("/api/users", getUsers)
	http.HandleFunc("/api/update", updateUser)
	
	port := "8080"
	fmt.Println("===========================================")
	fmt.Println("🚀 СЕРВЕР ДЛЯ TELEGRAM MINI APP ЗАПУЩЕН")
	fmt.Println("===========================================")
	fmt.Println("📡 Локальный адрес: http://localhost:" + port)
	fmt.Println("🤖 Готов к работе с Telegram")
	fmt.Println("===========================================")
	
	log.Fatal(http.ListenAndServe(":"+port, nil))
}