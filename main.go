package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings" // Pacote novo para manipular strings
)

// Estrutura para receber os dados da API de geolocalização
type GeoLocation struct {
	City    string `json:"city"`
	Country string `json:"country"`
	IP      string `json:"ip"`
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// --- INÍCIO DA MUDANÇA ---

		// 1. Tenta pegar o IP real do header X-Forwarded-For (padrão em proxies/PaaS)
		ipStr := r.Header.Get("X-Forwarded-For")

		// 2. Se o header estiver vazio, usa o r.RemoteAddr como fallback (para rodar localmente)
		if ipStr == "" {
			var err error
			ipStr, _, err = net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				// Tenta usar o RemoteAddr diretamente se SplitHostPort falhar
				ipStr = r.RemoteAddr
			}
		} else {
			// O header pode ter vários IPs (ex: "client, proxy1, proxy2"). O primeiro é o original.
			ips := strings.Split(ipStr, ",")
			ipStr = strings.TrimSpace(ips[0])
		}
		
		// --- FIM DA MUDANÇA ---

		// A partir daqui, o resto do código continua igual, mas usando o ipStr que descobrimos
		ip := net.ParseIP(ipStr)
		apiURL := ""

		if ip != nil && ip.IsLoopback() {
			apiURL = "https://get.geojs.io/v1/ip/geo.json"
		} else {
			apiURL = fmt.Sprintf("https://get.geojs.io/v1/ip/geo/%s.json", ipStr)
		}

		resp, err := http.Get(apiURL)
		if err != nil {
			http.Error(w, "Não foi possível contatar o serviço de geolocalização", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, "Não foi possível ler a resposta da API", http.StatusInternalServerError)
			return
		}

		var location GeoLocation
		json.Unmarshal(body, &location)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "<h1>Sua Localização Aproximada (baseada em IP)</h1>")
		fmt.Fprintf(w, "<p><strong>Endereço de IP Detectado:</strong> %s</p>", location.IP)
		fmt.Fprintf(w, "<p><strong>Cidade:</strong> %s</p>", location.City)
		fmt.Fprintf(w, "<p><strong>País:</strong> %s</p>", location.Country)
	})

	fmt.Println("Servidor iniciado em http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}