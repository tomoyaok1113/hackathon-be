package main

import (
	"encoding/json"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/joho/godotenv"
	"github.com/oklog/ulid"
	_ "github.com/oklog/ulid"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

type ResponseMessage struct {
	Id       string `json:"id"`
	FromName string `json:"fromName"`
	Point    int    `json:"point"`
	Message  string `json:"message"`
}
type ImportMessage struct {
	Id       string `json:"id"`
	FromName string `json:"fromName"`
	ToName   string `json:"toName"`
	Point    int    `json:"point"`
	Message  string `json:"message"`
}

func handlerMessage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	switch r.Method {
	case http.MethodGet:
		username := r.URL.Query().Get("fromname")
		rows, err := db.Query("SELECT id, fromname, point, message FROM messagelist WHERE toname=?", username)
		if err != nil {
			log.Printf("fail: db.Query, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		users := make([]ResponseMessage, 0)
		for rows.Next() {
			var u ResponseMessage
			if err := rows.Scan(&u.Id, &u.FromName, &u.Point, &u.Message); err != nil {
				log.Printf("fail: rows.Scan, %v\n", err)

				if err := rows.Close(); err != nil { // 500を返して終了するが、その前にrowsのClose処理が必要
					log.Printf("fail: rows.Close(), %v\n", err)
				}
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			users = append(users, u)
		}

		// ②-4
		bytes, err := json.Marshal(users)
		if err != nil {
			log.Printf("fail: json.Marshal, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(bytes)
	case http.MethodPost:
		var v ImportMessage
		t, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal([]byte(t), &v); err != nil {
			log.Printf("fail: json.Unmarshal, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if (v.FromName == "") || (v.ToName == "") {
			log.Println("fail: name is empty")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		id := strconv.FormatUint(ulid.Timestamp(time.Now()), 10)
		tx, err := db.Begin()
		if err != nil {
			log.Printf("fail:db.Begin, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err = tx.Exec("INSERT INTO messagelist (id,fromname,toname,point,message) VALUE (?,?,?,?,?)", id, v.FromName, v.ToName, v.Point, v.Message)
		if err != nil {
			tx.Rollback()
			log.Printf("fail: db.Prepare, %v\n", err)
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
