package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync/atomic"

	"github.com/sisoputnfrba/tp-golang/utils"
)

var config Config
var seguirCicloInstrucciones bool
var MotivoDesalojo string
var HayInterrupcion int32
var TidInterrupcion int
var pidEjecucion int
var tidEjecucion int

// var mutexDesalojo sync.Mutex

func main() {
	// Config

	err := utils.IniciarConfiguracion(&config)
	if err != nil {
		fmt.Println("Error al cargar la configuración del CPU:", err)
		return
	}

	// Logger
	slog.SetLogLoggerLevel(utils.LogLevels[config.LogLevel])
	utils.IniciarLogger()

	// Iniciar Cliente Memoria
	utils.IniciarCliente(config.IPMemory, "memoria", config.PortMemory)

	// Iniciar Servidor
	utils.RegistrarRutas(rutas)
	go func() {
		slog.Info("Se inicio el servidor de cpu", "puerto", config.Port)
		panic(http.ListenAndServe(":"+strconv.Itoa(config.Port), nil))
	}()

	// Iniciar Cliente Kernel
	utils.IniciarCliente(config.IPKernel, "kernel", config.PortKernel)

	select {}
}

/* Ejecutar proceso */
func ejecutarProceso(w http.ResponseWriter, r *http.Request) {
	// Recibir tid y pid
	queryParams := r.URL.Query()
	pidRecibido := queryParams.Get("PID")
	tidRecibido := queryParams.Get("TID")
	pid, _ := strconv.Atoi(pidRecibido)
	tid, _ := strconv.Atoi(tidRecibido)

	pidEjecucion = pid
	tidEjecucion = tid

	atomic.StoreInt32(&HayInterrupcion, 0)

	slog.Debug(" \n------------------------------------------ ")
	slog.Debug("Se entro a ejecutar proceso ", "pid", pid, "tid", tid)

	// Asignar registros
	registrosCPU = PedirContextoEjecucionMemoria(pid, tid)
	slog.Info(fmt.Sprintf("## TID: %d - Solicito Contexto Ejecución", tid))

	// Ejecutar
	seguirCicloInstrucciones = true
	for seguirCicloInstrucciones {
		if atomic.LoadInt32(&HayInterrupcion) == 1 { // Lectura segura
			slog.Debug("Se detectó interrupción")
			seguirCicloInstrucciones = false
		}

		// Fetch: Buscar instruccion a ejecutar
		instruccionAEjecutar := fetchInstruccion(pid, tid)

		// Decode
		instruccionInterpretada := decodeInstruccion(instruccionAEjecutar)

		// Execute
		ejecutarInstruccion(instruccionInterpretada, tid)

		//seguirCicloInstrucciones = manejarInterrupcion(HayInterrupcion, tid) // Si es true, se sigue ejecutando el proceso.
		slog.Debug("Valor de seguirCicloInstrucciones: ", "SeguirCiclo ", seguirCicloInstrucciones)
		registrosCPU.PC++ // Incrementar el program counter
	}
	MotivoDesalojo = obtenerMotivoDesalojo()

	EnviarContextoDesalojo(pid, tid, MotivoDesalojo)
	EnviarContextoEjecucionMemoria(pid, tid, registrosCPU)
	slog.Info(fmt.Sprintf("## TID: %d - Actualizo Contexto Ejecución", tid))
}

/* --------------- Peticiones Generales --------------- */
// Contexto de Desalojo (Kernel)
func EnviarContextoDesalojo(pid int, tid int, motivoDesalojo string) {

	body, err := json.Marshal(BodyDesalojo{
		PID:    pid,
		TID:    tid,
		Motivo: motivoDesalojo,
	})

	if err != nil {
		return
	}

	cliente := &http.Client{}
	url := fmt.Sprintf("http://%s:%d/kernel/contexto_desalojo", config.IPKernel, config.PortKernel)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
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

	slog.Debug("Se envio el contexto de desalojo al Kernel", "Motivo", motivoDesalojo)
}

// Syscall e Interrupt
func EnviarSyscall(nombreSyscall string, parametros map[string]interface{}) {
	body, err := json.Marshal(SyscallBodyRequest{
		Nombre_syscall:     nombreSyscall,
		Parametros_syscall: parametros,
	})
	if err != nil {
		return
	}

	cliente := &http.Client{}
	url := fmt.Sprintf("http://%s:%d/kernel/syscalls", config.IPKernel, config.PortKernel)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
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

	slog.Debug("Se envio la syscall con sus parametros")
}
func obtenerMotivoDesalojo() string {

	return MotivoDesalojo
}

func atenderInterrupcion(w http.ResponseWriter, r *http.Request) {
	// Recibir interrupcion
	queryParams := r.URL.Query()
	MotivoDesalojo = queryParams.Get("MOTIVO")
	TidInterrupcion, _ = strconv.Atoi(queryParams.Get("TID"))
	slog.Info("## Llega interrupcion al puerto Interrupt")

	slog.Debug("Se recibió una interrupción", "Motivo", MotivoDesalojo, "TID", TidInterrupcion)

	if atomic.LoadInt32(&HayInterrupcion) != 1 {
		slog.Debug("Se recibió una interrupción", "Motivo", MotivoDesalojo, "TID", TidInterrupcion)
		SetMotivoDesalojo(MotivoDesalojo)

		atomic.StoreInt32(&HayInterrupcion, 1) // Modificación segura
		slog.Debug("Interrupción recibida y establecida", "HayInterrupcion", atomic.LoadInt32(&HayInterrupcion))
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Interrupción recibida"))
}
