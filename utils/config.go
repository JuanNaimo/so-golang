package utils

import (
	"encoding/json"
	"fmt"
	"os"
)

func IniciarConfiguracion(config interface{}) error {
	file, err := os.Open("config.json")
	if err != nil {
		return fmt.Errorf("error al abrir el archivo de configuracion: %w", err)
	}
	defer file.Close() // TIP: defer ejecuta al terminar el main, entonces aca va a cerrar el archivo config antes de terminar

	decoder := json.NewDecoder(file)
	err = decoder.Decode(config)
	if err != nil {
		return fmt.Errorf("error al decodificar el archivo de configuracion: %w", err)
	}

	return nil
}
