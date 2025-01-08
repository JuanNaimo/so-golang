package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
)

/* --------------- MMU --------------- */
func CalcularMMU(direccionLogica uint32, base uint32, limite uint32) (uint32, error) {
	// Comprobar segmentation fault
	if direccionLogica+base > limite {
		return 0, fmt.Errorf("-->error en el calculo de la direccion fisica: Segmentation fault")
	}

	//Calcular MMU
	direccionFisica := base + direccionLogica
	slog.Debug("La direccion es: ", "Direccion Fisica", direccionFisica)
	return direccionFisica, nil
}

/* --------------- Contexto de Desalojo --------------- */
func EnviarContextoEjecucionMemoria(pid int, tid int, registrosCPU RegistroCPU) {

	body, err := json.Marshal(BodyContextoEjecucion{
		PID:       pid,
		TID:       tid,
		Registros: registrosCPU,
	})

	if err != nil {
		return
	}

	cliente := &http.Client{}
	url := fmt.Sprintf("http://%s:%d/memoria/depositar_contexto_ejecucion", config.IPMemory, config.PortMemory)
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

	slog.Debug("Se envio el contexto de ejecucion a memoria")
}

func PedirContextoEjecucionMemoria(pid int, tid int) RegistroCPU {
	//Pedir los registros a memoria
	slog.Debug("Dentro de pedir contexto ejecucion")

	var body RegistroCPU
	var registrosMemoria RegistroCPU
	cliente := &http.Client{}
	url := fmt.Sprintf("http://%s:%d/memoria/pedir_contexto_ejecucion", config.IPMemory, config.PortMemory)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		slog.Error("Error")
	}
	q := req.URL.Query()
	q.Add("PID", strconv.Itoa(pid))
	q.Add("TID", strconv.Itoa(tid))
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Content-Type", "application/json")
	respuesta, err := cliente.Do(req)
	if err != nil {
		slog.Error("Error")
	}
	// Verificar el código de estado de la respuesta
	if respuesta.StatusCode != http.StatusOK {
		slog.Error("Error")
	}

	bodyBytes, _ := io.ReadAll(respuesta.Body)

	err = json.Unmarshal(bodyBytes, &body)
	if err != nil {
		slog.Error("Error")
	}

	registrosMemoria.AX = body.AX
	registrosMemoria.BX = body.BX
	registrosMemoria.CX = body.CX
	registrosMemoria.DX = body.DX
	registrosMemoria.EX = body.EX
	registrosMemoria.FX = body.FX
	registrosMemoria.GX = body.GX
	registrosMemoria.HX = body.HX
	registrosMemoria.PC = body.PC
	registrosMemoria.Base = body.Base
	registrosMemoria.Limite = body.Limite

	slog.Debug("Se recibio el contexto de ejecucion de Memoria", "Registro AX", registrosMemoria.AX, "Base", registrosMemoria.Base, "Program Counter", registrosMemoria.PC)

	return registrosMemoria
}
