package main

import (
	"net/http"
	"sync"

	"github.com/sisoputnfrba/tp-golang/utils"
)

/* -------------- Config & Rutas -------------- */

// Estructura de configuración
type Config struct {
	Port               int    `json:"port"`
	IPMemory           string `json:"ip_memory"`
	PortMemory         int    `json:"port_memory"`
	IPCPU              string `json:"ip_cpu"`
	PortCPU            int    `json:"port_cpu"`
	SchedulerAlgorithm string `json:"sheduler_algorithm"`
	Quantum            int    `json:"quantum"`
	LogLevel           string `json:"log_level"`
}

// Rutas del HandlerFunction
var rutas = map[string]http.HandlerFunc{
	"/handshake/{modulo}":       utils.IniciarServer,
	"/kernel/syscalls":          AtenderSyscalls,
	"/kernel/contexto_desalojo": AtenderContextoDesalojo,
}

/* -------------- Estructuras -------------- */

// Estado de los procesos e hilos
type ESTADO int

const (
	New ESTADO = iota
	Ready
	Running
	Blocked
	Exit
)

// Estructura de Procesos
type PCB struct {
	PID           int
	TIDs          []int
	Mutex         []MUTEX
	Tamanio       int
	Estado        ESTADO
	Archivo       string
	PrioridadHilo int
	ContadorTID   int
}

// Estructura de Hilos
type TCB struct {
	TID       int
	Prioridad int
	Quantum   int
	PID       int
	Estado    ESTADO
}

type MUTEX struct {
	recurso         string // Identificador del recurso
	asignado        bool   // Indica si el mutex está tomado
	TIDhiloasignado int    // TID del hilo que tomó el mutex
	//	hiloAsignado   *TCB   // Hilo que tomó el mutex
	colaBloqueados *utils.Cola[TCB]
}

// Estructura de los Pedidos de Syscall
type SyscallBodyRequest struct {
	Nombre_syscall     string                 `json:"nombre_syscall"`
	Parametros_syscall map[string]interface{} `json:"parametros"`
}
type BodyDesalojo struct {
	PID    int    `json:"PID"`
	TID    int    `json:"TID"`
	Motivo string `json:"Motivo"`
}

type RequestPeticionHilo struct {
	PID              int    `json:"PID"`
	TID              int    `json:"TID"`
	RutaPseudocodigo string `json:"RutaPseudocodigo"`
}

// Mapa de Hilos en el que el indice es el tid a esperar su finalizacion
var MapaBlockedJoin = make(map[int]TCB)

/* -------------- Colas -------------- */

// Cola de New (Procesos)
var ColaNew *utils.Cola[PCB]

// Cola de Procesos en Ready (Procesos)
var ColaProcesos *utils.Cola[PCB]

// Cola de Ready (Hilos)
var ColaReady *utils.Cola[TCB]

// Cola de Blocked (Hilos)
var ColaBlocked *utils.Cola[TCB]

// Cola de Bloqueados de Join (Hilos)
var ColaBlockedJoin *utils.Cola[TCB]

// Cola de Exit (Hilos)
var ColaExit *utils.Cola[TCB]

/* -------------- Sincronizacion -------------- */
// Semaforo Condicional Mutex para acceder a la ColaNew
var mutexNew sync.Mutex

// Semaforo Condicional Mutex para acceder a la ColaProcesos y a la ColaReady
var mutexReady sync.Mutex

// Semaforo Condicional Mutex para la espera de Procesos
var mutexProcesos sync.Mutex

// Semaforo Condicional Mutex para desalojo por prioridad
var mutexPrioridad sync.Mutex

// Semaforo Condicional Mutex para controlar la liberacion de espacio
var mutexMemoria sync.Mutex

// Semaforo Condicional Mutex para acceder a la ColaExit (Hilos)
var mutexExit sync.Mutex

// Semaforo para la sincrinizacion entre finaliazacion de hilos y procesos
var mutexFinalizacion sync.Mutex

/* -------------------------------------------- */
