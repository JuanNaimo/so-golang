package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/sisoputnfrba/tp-golang/utils"
)

var config Config

// El dios de las variables
var EspacioMemoria []byte

func main() {
	// Config
	err := utils.IniciarConfiguracion(&config)
	if err != nil {
		fmt.Println("Error al cargar la configuración de Memoria:", err)
		return
	}

	// Logger
	slog.SetLogLoggerLevel(utils.LogLevels[config.LogLevel])
	utils.IniciarLogger()

	// Iniciar Clientes
	utils.IniciarCliente(config.IPFilesystem, "fileSystem", config.PortFilesystem)

	// Iniciar Servidor
	utils.RegistrarRutas(rutas)

	go func() {
		slog.Info("Se inicio el servidor de memoria", "puerto", config.Port)
		panic(http.ListenAndServe(":"+strconv.Itoa(config.Port), nil))
	}()

	EspacioMemoria = make([]byte, config.MemorySize)
	inicializarMemoria()

	select {}
}

/* -------------------- Funciones Generales -------------------- */
func inicializarMemoria() {

	switch scheme := config.Scheme; scheme {
	case "FIJAS":
		baseAcumulada := 0
		for i, particion := range config.Partitions {
			MapaParticiones[i] = &Particion{Ocupado: false, Base: uint32(baseAcumulada), Tamanio: uint32(particion)}
			slog.Debug("Memoria - Particion", "Base", baseAcumulada, "Tamanio", particion)
			baseAcumulada += particion
		}
		if baseAcumulada > config.MemorySize {
			slog.Error("Memoria - Los espacios de las particiones suman mas que el espacio total de memoria")
		}
	case "DINAMICAS":
		MapaParticiones[0] = &Particion{Ocupado: false, Base: 0, Tamanio: uint32(config.MemorySize)}
	}

	slog.Debug("Memoria - Se inicializo la Memoria")
}

/* ------------------------------------------------------------- */
