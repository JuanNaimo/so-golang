package main

import (
	"net/http"

	"github.com/sisoputnfrba/tp-golang/utils"
)

type Config struct {
	IPMemory   string `json:"ip_memory"`
	PortMemory int    `json:"port_memory"`
	IPKernel   string `json:"ip_kernel"`
	PortKernel int    `json:"port_kernel"`
	Port       int    `json:"port"`
	LogLevel   string `json:"log_level"`
}

var rutas = map[string]http.HandlerFunc{
	"/handshake/{modulo}": utils.IniciarServer,
	"/cpu/proceso":        ejecutarProceso,
	"/cpu/interrupt":      atenderInterrupcion,
}

/* --------------- Estructura de Registros --------------- */ // IMPORTANTE: Mover a utils porque comparte con memoria lo mismo
type RegistroCPU struct {
	PC     uint32
	AX     uint32
	BX     uint32
	CX     uint32
	DX     uint32
	EX     uint32
	FX     uint32
	GX     uint32
	HX     uint32
	Base   uint32
	Limite uint32
}

var registrosCPU RegistroCPU

/* --------------- Estructuras de peticiones (Body) --------------- */
type SyscallBodyRequest struct {
	Nombre_syscall     string                 `json:"nombre_syscall"`
	Parametros_syscall map[string]interface{} `json:"parametros"`
}

type BodyInstrucciones struct {
	Nombre_instruccion       string                 `json:"nombre_instruccion"`
	Parametros_instrucciones map[string]interface{} `json:"parametros"`
}

type BodyDesalojo struct {
	PID    int    `json:"PID"`
	TID    int    `json:"TID"`
	Motivo string `json:"Motivo"`
}

type BodyContextoEjecucion struct {
	PID       int         `json:"PID"`
	TID       int         `json:"TID"`
	Registros RegistroCPU `json:"Registros"`
}
