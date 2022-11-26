package main

import (
	"encoding/json"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/joho/godotenv"
	_ "github.com/oklog/ulid"
	"io"
	"log"
	"net/http"
)

type FixMessage struct {
	Id      string `json:"id"`
	Fixname string `json:"fixname"`
	ToName  string `json:"toName"`
	Point   int    `json:"point"`
	Message string `json:"message"`
}

func handlerFixMessage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	switch r.Method {
	case http.MethodPost:
		var v FixMessage
		t, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal([]byte(t), &v); err != nil {
			log.Printf("fail: json.Unmarshal, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		tx, err := db.Begin()
		if err != nil {
			log.Printf("fail:db.Begin, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		messagepoint := 0
		if err := tx.QueryRow("SELECT point FROM messagelist WHERE id = ?", v.Id).Scan(&messagepoint); err != nil {
			tx.Rollback()

			log.Printf("fail: db.messagepoint, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		log.Printf("%v\n", messagepoint)
		userpoint := 0
		if err := tx.QueryRow("SELECT point FROM userlist WHERE name = ?", v.Fixname).Scan(&userpoint); err != nil {
			tx.Rollback()
			log.Printf("fail: db.userpoint, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		log.Printf("%v\n", userpoint)
		userpoint = userpoint - messagepoint
		_, err = tx.Exec("UPDATE userlist SET point=? WHERE name=?", userpoint, v.Fixname)
		_, err = tx.Exec("UPDATE messagelist SET toname=?, point=?, message=? WHERE id=?", v.ToName, v.Point, v.Message, v.Id)
		if err != nil {
			tx.Rollback()
			log.Printf("fail: db.updatemessagelist, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if err := tx.Commit(); err != nil {
			log.Printf("fail:tx.Commit, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if err != nil {
			log.Printf("fail: json.Marshal, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	default:
		log.Printf("fail: HTTP Method is %s\n", r.Method)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}
