package main

import (
	"encoding/json"
	"fmt"
	"io" // <--- MUDANÇA: Usamos 'io' em vez de 'io/ioutil'
	"log"
	"net"
	"net/http"
)

// Estrutura para receber os dados da API de geolocalização
type GeoLocation struct {
	City    string `json:"city"`
	Country string `json:"country"`
	IP      string `json:"ip"`
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Pega o endereço de IP do visitante
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)

		// Chama a API de geolocalização com o IP do visitante
		apiURL := fmt.Sprintf("https://get.geojs.io/v1/ip/geo/%s.json", ip)
		resp, err := http.Get(apiURL)
		if err != nil {
			http.Error(w, "Não foi possível contatar o serviço de geolocalização", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		// MUDANÇA: Usamos io.ReadAll em vez de ioutil.ReadAll
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, "Não foi possível ler a resposta da API", http.StatusInternalServerError)
			return
		}

		var location GeoLocation
		json.Unmarshal(body, &location)

		// Escreve a resposta HTML diretamente para o navegador
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "<h1>Sua Localização Aproximada (baseada em IP)</h1>")
		fmt.Fprintf(w, "<p><strong>Endereço de IP:</strong> %s</p>", location.IP)
		fmt.Fprintf(w, "<p><strong>Cidade:</strong> %s</p>", location.City)
		fmt.Fprintf(w, "<p><strong>País:</strong> %s</p>", location.Country)
	})

	fmt.Println("Servidor iniciado em http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}