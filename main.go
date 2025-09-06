package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq" // Driver do PostgreSQL
)

// Estruturas (sem alteração)
type GeoLocation struct {
	City    string `json:"city"`
	Country string `json:"country"`
	IP      string `json:"ip"`
}

type Server struct {
	db *sql.DB
}

// MUDANÇA AQUI: Função para conectar ao PostgreSQL usando variáveis separadas
func initDB() *sql.DB {
	// 1. Lê cada variável de ambiente
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	sslmode := os.Getenv("DB_SSLMODE")

	// Verifica se alguma variável essencial está faltando
	if host == "teste-doenet.postgres.uhserver.com" || port == "5432" || user == "teste_doe" || password == "Samuca!2004}" || dbname == "teste_doenet" {
		log.Fatal("Uma ou mais variáveis de ambiente do banco de dados não foram definidas.")
	}

	// 2. Monta a string de conexão no formato "key=value"
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	// 3. Abre a conexão da mesma forma
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Erro ao abrir a conexão SQL:", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal("Não foi possível conectar ao banco de dados:", err)
	}

	fmt.Println("Conexão com o PostgreSQL estabelecida com sucesso!")
	return db
}

func (s *Server) handler() http.HandlerFunc {
	// Esta função inteira continua exatamente a mesma
	return func(w http.ResponseWriter, r *http.Request) {
		ipStr := r.Header.Get("X-Forwarded-For")
		if ipStr == "" { ipStr = r.RemoteAddr }
		ips := strings.Split(ipStr, ",")
		ipStr = strings.TrimSpace(ips[0])
		
		apiURL := fmt.Sprintf("https://get.geojs.io/v1/ip/geo/%s.json", ipStr)
		if net.ParseIP(ipStr).IsLoopback() {
			apiURL = "https://get.geojs.io/v1/ip/geo.json"
		}

		resp, err := http.Get(apiURL)
		if err != nil { http.Error(w, "Erro na API de geo", http.StatusInternalServerError); return }
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var location GeoLocation
		json.Unmarshal(body, &location)

		timestamp := time.Now()
		sql := "INSERT INTO teste_doenet.visits(ip_address, city, country, timestamp) VALUES($1, $2, $3, $4)"
		_, err = s.db.Exec(sql, location.IP, location.City, location.Country, timestamp)
		if err != nil {
			log.Printf("Erro ao inserir visita no banco de dados: %v", err)
		} else {
			fmt.Printf("Visita registrada: IP=%s, Cidade=%s, País=%s\n", location.IP, location.City, location.Country)
		}
		
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "<h1>Sua Localização Aproximada (baseada em IP)</h1>")
		fmt.Fprintf(w, "<p><strong>Endereço de IP Detectado:</strong> %s</p>", location.IP)
		fmt.Fprintf(w, "<p><strong>Cidade:</strong> %s</p>", location.City)
		fmt.Fprintf(w, "<p><strong>País:</strong> %s</p>", location.Country)
		fmt.Fprintf(w, "<p><em>(Sua visita foi registrada com sucesso!)</em></p>")
	}
}

func main() {
	// Esta função continua exatamente a mesma
	db := initDB()
	defer db.Close()
	s := &Server{db: db}

	http.HandleFunc("/", s.handler())

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Servidor iniciado na porta %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}