package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/joho/godotenv"
	"github.com/oklog/ulid"
	_ "github.com/oklog/ulid"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

type ResponseUser struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Point int    `json:"point"`
}

type ImportUser struct {
	Name  string `json:"name"`
	Point int    `json:"point"`
}

// ① GoプログラムからMySQLへ接続
var db *sql.DB

func init() {
	mysqlUser := os.Getenv("MYSQL_USER")
	mysqlPwd := os.Getenv("MYSQL_PWD")
	mysqlHost := os.Getenv("MYSQL_HOST")
	mysqlDatabase := os.Getenv("MYSQL_DATABASE")

	// ①-2
	connStr := fmt.Sprintf("%s:%s@%s/%s", mysqlUser, mysqlPwd, mysqlHost, mysqlDatabase)
	_db, err := sql.Open("mysql", connStr)
	if err != nil {
		log.Fatalf("fail: sql.Open, %v\n", err)
	}
	// ①-3
	if err := _db.Ping(); err != nil {
		log.Fatalf("fail: _db.Ping, %v\n", err)
	}
	db = _db
}

// ② /userでリクエストされたらnameパラメーターと一致する名前を持つレコードをJSON形式で返す
func handlerUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	switch r.Method {
	case http.MethodGet:
		rows, err := db.Query("SELECT id, name, point FROM userlist")
		if err != nil {
			log.Printf("fail: db.Query, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		users := make([]ResponseUser, 0)
		for rows.Next() {
			var u ResponseUser
			if err := rows.Scan(&u.Id, &u.Name, &u.Point); err != nil {
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
		var v ImportUser
		t, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal([]byte(t), &v); err != nil {
			log.Printf("fail: json.Unmarshal, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if v.Name == "" {
			log.Println("fail: name is empty")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if len(v.Name) >= 50 {
			log.Println("fail: name is too long")
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
		_, err = tx.Exec("INSERT INTO userlist (id,name,point) VALUE (?,?,?)", id, v.Name, v.Point)
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

func main() {
	http.HandleFunc("/user", handlerUser)
	http.HandleFunc("/message", handlerMessage)
	http.HandleFunc("/messagesub", handlerMessageForSubmitlist)
	http.HandleFunc("/point", handlerPoint)
	http.HandleFunc("/messagedelete", handlerDeleteMessage)
	http.HandleFunc("/messagefix", handlerFixMessage)

	// ③ Ctrl+CでHTTPサーバー停止時にDBをクローズする
	closeDBWithSysCall()
	// 8000番ポートでリクエストを待ち受ける
	log.Println("Listening...")
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatal(err)
	}
}

// ③ Ctrl+CでHTTPサーバー停止時にDBをクローズする
func closeDBWithSysCall() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		s := <-sig
		log.Printf("received syscall, %v", s)

		if err := db.Close(); err != nil {
			log.Fatal(err)
		}
		log.Printf("success: db.Close()")
		os.Exit(0)
	}()
}
