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

type AddPoint struct {
	ToName string `json:"toName"`
	Point  int    `json:"point"`
}

func handlerPoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	switch r.Method {
	case http.MethodGet:
		username := r.URL.Query().Get("username")
		point, err := db.Query("SELECT point FROM userlist WHERE name=?", username)
		if err != nil {
			log.Printf("fail: db.Query, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// ②-4
		bytes, err := json.Marshal(point)
		if err != nil {
			log.Printf("fail: json.Marshal, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(bytes)
	case http.MethodPost:
		var v AddPoint
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
		_, err = tx.Exec("UPDATE userlist SET point+=v.Point WHERE name=v.ToName")
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
