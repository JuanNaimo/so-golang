package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

func IniciarServer(w http.ResponseWriter, r *http.Request) {
	modulo := r.PathValue("modulo")

	respuesta, err := json.Marshal(fmt.Sprintf("Respuesta del modulo %s", modulo))
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError) // TODO: Cambiar por log
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

func IniciarCliente(ip string, moduloNombre string, puerto int) {
	cliente := &http.Client{}
	url := fmt.Sprintf("http://%s:%d/handshake/%s", ip, puerto, moduloNombre)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")

	seEncontroServer := true
	var respuesta *http.Response
	for seEncontroServer {
		respuesta, err = cliente.Do(req)
		slog.Debug("Buscando conectar con el servidor -", "servidor", moduloNombre)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		if respuesta.StatusCode == http.StatusNotFound {
			time.Sleep(1 * time.Second)
			continue
		} else {
			seEncontroServer = false
		}
	}

	// Verificar el código de estado de la respuesta
	if respuesta.StatusCode != http.StatusOK {
		return
	}

	bodyBytes, err := io.ReadAll(respuesta.Body)
	if err != nil {
		return
	}

	slog.Info(string(bodyBytes), "status_code", http.StatusOK)
}

func RegistrarRutas(rutas map[string]http.HandlerFunc) {
	for ruta, handler := range rutas {
		http.HandleFunc(ruta, handler)
	}
}
