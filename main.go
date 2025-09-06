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

// Estrutura para os dados da API de geolocalização
type GeoLocation struct {
	City    string `json:"city"`
	Country string `json:"country"`
	IP      string `json:"ip"`
}

// Estrutura para nosso servidor
type Server struct {
	db *sql.DB
}

// Função para conectar ao PostgreSQL
func initDB() *sql.DB {
	// Pega a URL de conexão da variável de ambiente (DATABASE_URL)
	connStr := os.Getenv("postgresql://teste_doe:S@muc@2004@teste-doenet.postgres.uhserver.com:5432/teste_doenet")
	if connStr == "" {
		log.Fatal("A variável de ambiente DATABASE_URL não está definida.")
	}

	// Abre a conexão com o banco de dados PostgreSQL
	db, err := sql.Open("postgres", connStr)
	// A CORREÇÃO ESTÁ AQUI:
	if err != nil {
		log.Fatal(err)
	}

	// Testa a conexão para garantir que tudo está OK
	err = db.Ping()
	if err != nil {
		log.Fatal("Não foi possível conectar ao banco de dados:", err)
	}

	fmt.Println("Conexão com o PostgreSQL estabelecida com sucesso!")
	
	return db
}

func (s *Server) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// --- 1. Obter a localização ---
		ipStr := r.Header.Get("X-Forwarded-For")
		if ipStr == "" {
			ipStr = r.RemoteAddr
		}
		ips := strings.Split(ipStr, ",")
		ipStr = strings.TrimSpace(ips[0])
		
		apiURL := fmt.Sprintf("https://get.geojs.io/v1/ip/geo/%s.json", ipStr)
		if net.ParseIP(ipStr).IsLoopback() {
			apiURL = "https://get.geojs.io/v1/ip/geo.json"
		}

		resp, err := http.Get(apiURL)
		if err != nil {
			http.Error(w, "Erro na API de geo", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var location GeoLocation
		json.Unmarshal(body, &location)

		// --- 2. Salvar a visita no banco de dados ---
		timestamp := time.Now()
		
		sql := "INSERT INTO visits(ip_address, city, country, timestamp) VALUES($1, $2, $3, $4)"
		_, err = s.db.Exec(sql, location.IP, location.City, location.Country, timestamp)
		if err != nil {
			log.Println("Erro ao inserir visita no banco de dados:", err)
		} else {
			fmt.Printf("Visita registrada: IP=%s, Cidade=%s, País=%s\n", location.IP, location.City, location.Country)
		}
		
		// --- 3. Mostrar a localização para o usuário ---
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
		port = "8080" // Porta padrão para rodar localmente
	}

	fmt.Printf("Servidor iniciado na porta %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}