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

// MUDANÇA AQUI: As credenciais estão diretamente no código
func initDB() *sql.DB {

	// ATENÇÃO: DADOS SENSÍVEIS DIRETAMENTE NO CÓDIGO (NÃO RECOMENDADO!)
	host := "teste-doenet.postgres.uhserver.com"
	port := "5432"
	user := "teste_doe"
	password := "Samuca!2004}" // <-- SUA SENHA FICA EXPOSTA AQUI!
	dbname := "teste_doenet"
	sslmode := "require"

	// Monta a string de conexão a partir das variáveis acima
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

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
	// Esta função continua exatamente a mesma de antes
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
	db := initDB()
	defer db.Close()
	s := &Server{db: db}

	http.HandleFunc("/", s.handler())

	// Não precisamos mais ler a PORT do Render se quisermos fixar, mas é boa prática manter
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Servidor iniciado na porta %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}