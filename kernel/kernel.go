package main

import (
	"fmt"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"

	"github.com/sisoputnfrba/tp-golang/utils"
)

var config Config
var pidContador = -1

var pcbEjecutando PCB
var tcbEjecutando TCB

func main() {
	// Config
	err := utils.IniciarConfiguracion(&config)
	if err != nil {
		fmt.Println("Error al cargar la configuración del kernel:", err)
		return
	}

	// Logs
	slog.SetLogLoggerLevel(utils.LogLevels[config.LogLevel])
	utils.IniciarLogger()

	// Iniciar Colas y Semaforos
	ColaNew = &utils.Cola[PCB]{}
	ColaProcesos = &utils.Cola[PCB]{}
	ColaReady = &utils.Cola[TCB]{}
	ColaBlockedJoin = &utils.Cola[TCB]{}
	ColaExit = &utils.Cola[TCB]{}

	// Iniciar Clientes
	utils.IniciarCliente(config.IPCPU, "cpu", config.PortCPU)
	utils.IniciarCliente(config.IPMemory, "memoria", config.PortMemory)

	// Iniciar Servidor
	utils.RegistrarRutas(rutas)
	go func() {
		slog.Info("Se inicio el servidor de kernel", "puerto", config.Port)
		slog.Debug("----------------------------------------------------")
		panic(http.ListenAndServe(":"+strconv.Itoa(config.Port), nil))
	}()

	// Iniciar Planificadores
	go PlanificadorCortoPlazo()

	go PlanificadorLargoPlazo()

	go SeleccionarHiloPrioridad()

	// Argumento del Proceso Inicial
	args := os.Args    // args[0] es el nombre del programa, hay que usar a partir del [1]
	if len(args) > 2 { // Mayor a 2 porque yo le estoy pasando mas de 2 cosas
		nombreArchivo := args[1]
		slog.Debug("Archivo ingresado por argumento", "Archivo", nombreArchivo)
		tamanioProceso := args[2]
		tamanio, _ := strconv.Atoi(tamanioProceso)
		slog.Debug("Se ingresaron los argumentos correspondientes - ", "Nombre del Archivo", nombreArchivo, "Tamaño del Proceso", tamanio)
		CrearProceso(nombreArchivo, tamanio, 0)
	} else {
		slog.Error("No se le pasaron argumentos al kernel")
	}

	select {}
}
