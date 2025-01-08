package main

import (
	"net/http"

	"github.com/sisoputnfrba/tp-golang/utils"
)

type Config struct {
	Port            int    `json:"port"`
	MemorySize      int    `json:"memory_size"`
	InstructionPath string `json:"instruction_path"`
	ResponseDelay   int    `json:"response_delay"`
	IPKernel        string `json:"ip_kernel"`
	PortKernel      int    `json:"port_kernel"`
	IPCPU           string `json:"ip_cpu"`
	PortCPU         int    `json:"port_cpu"`
	IPFilesystem    string `json:"ip_filesystem"`
	PortFilesystem  int    `json:"port_filesystem"`
	Scheme          string `json:"scheme"`
	SearchAlgorithm string `json:"search_algorithm"`
	Partitions      []int  `json:"partitions"`
	LogLevel        string `json:"log_level"`
}

var rutas = map[string]http.HandlerFunc{
	"/handshake/{modulo}":                   utils.IniciarServer,
	"/memoria/finalizarproceso":             FinalizarProcesoMemoria,
	"/memoria/crearproceso/{pid}":           CrearProcesoMemoria,
	"/memoria/finalizarhilo":                FinalizarHiloMemoria,
	"/memoria/crearhilo":                    CrearHiloMemoria,
	"/memoria/pedido_instrucciones":         AtenderPedidoInstrucciones,
	"/memoria/ejecutar_instruccion":         AtenderEjecutarInstruccion,
	"/memoria/depositar_contexto_ejecucion": GuardarContextoEjecucion,
	"/memoria/pedir_contexto_ejecucion":     DevolverContextoEjecucion,
	"/memoria/dumpmemory":                   AtenderDumpMemory,
}

/* --------------- Estructuras de peticiones (Body) --------------- */
type RequestPeticionHilo struct {
	PID              int    `json:"PID"`
	TID              int    `json:"TID"`
	RutaPseudocodigo string `json:"RutaPseudocodigo"`
}
type RequestPeticion struct {
	PID              int    `json:"PID"`
	TamanioProceso   int    `json:"TamanioProceso"`
	RutaPseudocodigo string `json:"RutaPseudocodigo"`
}

type RequestFinalizarHilo struct {
	PID int `json:"PID"`
	TID int `json:"TID"`
}

type BodyInstrucciones struct {
	Nombre_instruccion       string                 `json:"nombre_instruccion"`
	Parametros_instrucciones map[string]interface{} `json:"parametros"`
}

type BodyContextoEjecucion struct {
	PID       int             `json:"PID"`
	TID       int             `json:"TID"`
	Registros BodyRegistroCPU `json:"Registros"`
}

/* --------------- Estructuras Generales --------------- */

type RegistroCPU struct {
	PC uint32
	AX uint32
	BX uint32
	CX uint32
	DX uint32
	EX uint32
	FX uint32
	GX uint32
	HX uint32
}

// Es solo para desempaquetar el envio de cpu
type BodyRegistroCPU struct {
	PC     uint32 `json:"PC"`
	AX     uint32 `json:"AX"`
	BX     uint32 `json:"BX"`
	CX     uint32 `json:"CX"`
	DX     uint32 `json:"DX"`
	EX     uint32 `json:"EX"`
	FX     uint32 `json:"FX"`
	GX     uint32 `json:"GX"`
	HX     uint32 `json:"HX"`
	Base   uint32 `json:"Base"`
	Limite uint32 `json:"Limite"`
}

type PidYTid struct {
	PID int
	TID int
}

type BaseYLimite struct {
	PunteroParticion *Particion
	Base             uint32
	Limite           uint32
}

type Particion struct {
	Ocupado bool
	Base    uint32
	Tamanio uint32
}

type BodyDumpFile struct {
	FileName string
	Size     int
	Content  []byte
}

/* --------------- Mapas --------------- */
// Mapa que relaciona TID con Array de Instrucciones (lineas de seudocodigo)
var MapaInstrucciones = make(map[PidYTid][]string)

// Mapa de Pid que devuelve BaseYLimite
var MapaPidContexto = make(map[int]BaseYLimite)

// Mapa para que tid que recibe PidYTid y que devuelve Registros
var MapaTidContexto = make(map[PidYTid]RegistroCPU)

// Mapa de particiones en el que el int seria el segmento							((!!!!))Creo que es redundante con MapaPidContexto
var MapaParticiones = make(map[int]*Particion)

/* ------------------------------------- */
