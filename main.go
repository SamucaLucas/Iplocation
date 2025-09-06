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

// Função para conectar ao PostgreSQL
func initDB() *sql.DB {
	connStr := os.Getenv("postgres://teste_doe:Samuca!2004}@teste-doenet.postgres.uhserver.com:5432/teste_doenet")
	if connStr == "" {
		log.Fatal("A variável de ambiente DATABASE_URL não está definida.")
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Erro ao abrir a conexão SQL:", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal("Não foi possível conectar ao banco de dados. Verifique a URL e o firewall:", err)
	}

	fmt.Println("Conexão com o PostgreSQL estabelecida com sucesso!")
	return db
}

func (s *Server) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// --- Obtenção da localização (sem alteração) ---
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

		// --- Salvar a visita no banco de dados com o SCHEMA CORRETO ---
		timestamp := time.Now()
		
		// A CORREÇÃO ESTÁ AQUI:
		sql := "INSERT INTO teste_doenet.visits(ip_address, city, country, timestamp) VALUES($1, $2, $3, $4)"
		
		_, err = s.db.Exec(sql, location.IP, location.City, location.Country, timestamp)
		if err != nil {
			// Este log agora será mais útil para depurar
			log.Printf("Erro ao inserir visita no banco de dados (schema: teste_doenet): %v", err)
		} else {
			fmt.Printf("Visita registrada: IP=%s, Cidade=%s, País=%s\n", location.IP, location.City, location.Country)
		}
		
		// --- Mostrar a localização para o usuário (sem alteração) ---
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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Servidor iniciado na porta %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}