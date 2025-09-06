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

// initDB (sem alteração, usando a versão com credenciais no código)
func initDB() *sql.DB {
	host := "teste-doenet.postgres.uhserver.com"
	port := "5432"
	user := "teste_doe"
	password := "Samuca!2004}"
	dbname := "teste_doenet"
	sslmode := "disable"

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

// MUDANÇA: O handler agora executa a lógica E serve o arquivo HTML
func (s *Server) mainPageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// --- 1. Lógica de coleta de localização (executa nos bastidores) ---
		go func() { // Usamos uma goroutine para não atrasar o carregamento da página
			ipStr := r.Header.Get("X-Forwarded-For")
			if ipStr == "" { ipStr = r.RemoteAddr }
			ips := strings.Split(ipStr, ",")
			ipStr = strings.TrimSpace(ips[0])
			
			apiURL := fmt.Sprintf("https://get.geojs.io/v1/ip/geo/%s.json", ipStr)
			if net.ParseIP(ipStr).IsLoopback() {
				apiURL = "https://get.geojs.io/v1/ip/geo.json"
			}

			resp, err := http.Get(apiURL)
			if err != nil {
				log.Printf("Erro na API de geo: %v", err)
				return
			}
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
		}()

		// --- 2. Serve a página principal para o usuário ---
		http.ServeFile(w, r, "./static/index.html")
	}
}

func main() {
	db := initDB()
	defer db.Close()
	s := &Server{db: db}

	// MUDANÇA: Criamos um "FileServer" para a pasta "static"
	// Isso faz com que o CSS, imagens, etc., sejam carregados corretamente
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// MUDANÇA: O handler principal agora é a função que serve a página
	http.HandleFunc("/", s.mainPageHandler())

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Servidor iniciado na porta %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}