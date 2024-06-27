package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Cotacao struct {
	USDBRL struct {
		Code       string `json:"code"`
		Codein     string `json:"codein"`
		Name       string `json:"name"`
		High       string `json:"high"`
		Low        string `json:"low"`
		VarBid     string `json:"varBid"`
		PctChange  string `json:"pctChange"`
		Bid        string `json:"bid"`
		Ask        string `json:"ask"`
		Timestamp  string `json:"timestamp"`
		CreateDate string `json:"create_date"`
	} `json:"USDBRL"`
}

func main() {
	httpClient := &http.Client{}
	os.Remove("./sqlite-database.db")

	db, err := sql.Open("sqlite3", "./sqlite-database.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	criarTabela(db)

	mux := http.NewServeMux()
	mux.HandleFunc("/cotacao", func(w http.ResponseWriter, r *http.Request) {
		getCotacao(w, httpClient, db)
	})
	http.ListenAndServe(":8080", mux)
}

func getCotacao(w http.ResponseWriter, httpClient *http.Client, db *sql.DB) {

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	req, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	if err != nil {
		panic(err)
	}

	response, err := httpClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer cancel()

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	var cotacao Cotacao
	json.Unmarshal([]byte(body), &cotacao)
	fmt.Println(cotacao)
	inserirCotacao(db, &cotacao)
	w.Write([]byte(cotacao.USDBRL.Bid))
}

func criarTabela(db *sql.DB) {
	stmt, err := db.Prepare(`create table cotacoes (
		"code" TEXT,
		"codein" TEXT,
		"name"  TEXT,
		"high"  TEXT,
		"low"   TEXT,
		"varbid"   TEXT,
		"pctChange"   TEXT,
		"bid"   TEXT,
		"ask"    TEXT,
		"timestamp"   TEXT,
		"create_date"   TEXT)`)
	if err != nil {
		panic(err)
	}
	defer stmt.Close()
	_, err = stmt.Exec()
	if err != nil {
		panic(err)
	}
}

func inserirCotacao(db *sql.DB, cotacao *Cotacao) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
	defer cancel()
	stmt, err := db.Prepare(`insert into cotacoes (
		code, 
		codein, 
		name, 
		high, 
		low, 
		varbid, 
		pctChange, 
		bid, 
		ask, 
		timestamp, 
		create_date
		) values(?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		panic(err)
	}
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx, cotacao.USDBRL.Code,
		cotacao.USDBRL.Codein,
		cotacao.USDBRL.Name,
		cotacao.USDBRL.High,
		cotacao.USDBRL.Low,
		cotacao.USDBRL.VarBid,
		cotacao.USDBRL.PctChange,
		cotacao.USDBRL.Bid,
		cotacao.USDBRL.Ask,
		cotacao.USDBRL.Timestamp,
		cotacao.USDBRL.CreateDate)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			fmt.Printf("DB operation timed out: %v\n", err)
			return
		}
		var netError net.Error
		if errors.As(err, &netError) && netError.Timeout() {
			fmt.Printf("network timed out: %v\n", err)
			return
		}
		fmt.Println("unknow error")
		return
	}
	fmt.Println("Cotacao salva!")
}
