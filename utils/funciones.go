package utils

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

// IDEA PARA HACER FUNCION GENERAL PARA PETICIONES
func EnviarPeticion(metodo string, ip string, endpoint string, puerto int, cuerposol any) {

	cliente := &http.Client{}
	url := fmt.Sprintf("http://%s:%d/%s", ip, puerto, endpoint)
	req, err := http.NewRequest(metodo, url, nil)
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")
	respuesta, err := cliente.Do(req)
	if err != nil {
		return
	}

	// Verificar el código de estado de la respuesta
	if respuesta.StatusCode != http.StatusOK {
		return
	}

	bodyBytes, err := io.ReadAll(respuesta.Body)
	if err != nil {
		return
	}

	slog.Info(string(bodyBytes))

	fmt.Println(string(bodyBytes))
}
