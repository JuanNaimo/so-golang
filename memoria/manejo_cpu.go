package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

/* --------------- Manejo de Instrucciones y Desalojo de CPU --------------- */
func AtenderPedidoInstrucciones(w http.ResponseWriter, r *http.Request) {

	queryParams := r.URL.Query()
	pidRecibido := queryParams.Get("PID")
	tidRecibido := queryParams.Get("TID")
	PCrecibido := queryParams.Get("PC")
	pid, _ := strconv.Atoi(pidRecibido)
	tid, _ := strconv.Atoi(tidRecibido)
	pc, _ := strconv.Atoi(PCrecibido)
	//Sumarle 1 al PC	(consigna)
	pc++
	instruccionesDelHilo := MapaInstrucciones[PidYTid{PID: pid, TID: tid}]
	//slog.Debug("Instrucciones del hilo", "Instrucciones", instruccionesDelHilo)
	instruccionAdevolver := instruccionesDelHilo[pc]
	var cuerpoinstrucciones BodyInstrucciones

	//slog.Debug("Memoria Cpu - Se ingreso al pedido de instrucciones", "TID", tid)

	delimitador := " "
	partesInstruccion := strings.Split(instruccionAdevolver, delimitador)

	switch partesInstruccion[0] {
	case "SET":
		cuerpoinstrucciones.Nombre_instruccion = "SET"
		cuerpoinstrucciones.Parametros_instrucciones = make(map[string]interface{})
		cuerpoinstrucciones.Parametros_instrucciones["Registro"] = partesInstruccion[1]
		cuerpoinstrucciones.Parametros_instrucciones["Valor"] = partesInstruccion[2]

	case "SUM":
		cuerpoinstrucciones.Nombre_instruccion = "SUM"
		cuerpoinstrucciones.Parametros_instrucciones = make(map[string]interface{})

		cuerpoinstrucciones.Parametros_instrucciones["RegistroDestino"] = partesInstruccion[1]
		cuerpoinstrucciones.Parametros_instrucciones["RegistroOrigen"] = partesInstruccion[2]

	case "SUB":
		cuerpoinstrucciones.Nombre_instruccion = "SUB"
		cuerpoinstrucciones.Parametros_instrucciones = make(map[string]interface{})

		cuerpoinstrucciones.Parametros_instrucciones["RegistroDestino"] = partesInstruccion[1]
		cuerpoinstrucciones.Parametros_instrucciones["RegistroOrigen"] = partesInstruccion[2]

	case "JNZ":
		cuerpoinstrucciones.Nombre_instruccion = "JNZ"
		cuerpoinstrucciones.Parametros_instrucciones = make(map[string]interface{})

		cuerpoinstrucciones.Parametros_instrucciones["Registro"] = partesInstruccion[1]
		cuerpoinstrucciones.Parametros_instrucciones["Instruccion"] = partesInstruccion[2]

	case "LOG":
		cuerpoinstrucciones.Nombre_instruccion = "LOG"
		cuerpoinstrucciones.Parametros_instrucciones = make(map[string]interface{})

		cuerpoinstrucciones.Parametros_instrucciones["Registro"] = partesInstruccion[1]

	case "WRITE_MEM":
		cuerpoinstrucciones.Nombre_instruccion = "WRITE_MEM"
		cuerpoinstrucciones.Parametros_instrucciones = make(map[string]interface{})

		cuerpoinstrucciones.Parametros_instrucciones["RegistroDireccion"] = partesInstruccion[1]
		cuerpoinstrucciones.Parametros_instrucciones["RegistroDatos"] = partesInstruccion[2]

	case "READ_MEM":
		cuerpoinstrucciones.Nombre_instruccion = "READ_MEM"
		cuerpoinstrucciones.Parametros_instrucciones = make(map[string]interface{})

		cuerpoinstrucciones.Parametros_instrucciones["RegistroDatos"] = partesInstruccion[1]
		cuerpoinstrucciones.Parametros_instrucciones["RegistroDireccion"] = partesInstruccion[2]

	case "DUMP_MEMORY":
		cuerpoinstrucciones.Nombre_instruccion = "DUMP_MEMORY"
		cuerpoinstrucciones.Parametros_instrucciones = make(map[string]interface{})

		cuerpoinstrucciones.Parametros_instrucciones = nil

	case "IO":
		cuerpoinstrucciones.Nombre_instruccion = "IO"
		cuerpoinstrucciones.Parametros_instrucciones = make(map[string]interface{})

		cuerpoinstrucciones.Parametros_instrucciones["Tiempo"] = partesInstruccion[1]

	case "PROCESS_CREATE":
		cuerpoinstrucciones.Nombre_instruccion = "PROCESS_CREATE"
		cuerpoinstrucciones.Parametros_instrucciones = make(map[string]interface{})
		cuerpoinstrucciones.Parametros_instrucciones["ArchivoInstrucciones"] = partesInstruccion[1]
		cuerpoinstrucciones.Parametros_instrucciones["Tamanio"] = partesInstruccion[2]
		cuerpoinstrucciones.Parametros_instrucciones["Prioridad"] = partesInstruccion[3]

	case "THREAD_CREATE":
		cuerpoinstrucciones.Nombre_instruccion = "THREAD_CREATE"
		cuerpoinstrucciones.Parametros_instrucciones = make(map[string]interface{})
		cuerpoinstrucciones.Parametros_instrucciones["ArchivoInstrucciones"] = partesInstruccion[1]
		cuerpoinstrucciones.Parametros_instrucciones["Prioridad"] = partesInstruccion[2]

	case "THREAD_JOIN":
		cuerpoinstrucciones.Nombre_instruccion = "THREAD_JOIN"
		cuerpoinstrucciones.Parametros_instrucciones = make(map[string]interface{})
		cuerpoinstrucciones.Parametros_instrucciones["Tid"] = partesInstruccion[1]

	case "THREAD_CANCEL":
		cuerpoinstrucciones.Nombre_instruccion = "THREAD_CANCEL"
		cuerpoinstrucciones.Parametros_instrucciones = make(map[string]interface{})
		cuerpoinstrucciones.Parametros_instrucciones["Tid"] = partesInstruccion[1]

	case "MUTEX_CREATE":
		cuerpoinstrucciones.Nombre_instruccion = "MUTEX_CREATE"
		cuerpoinstrucciones.Parametros_instrucciones = make(map[string]interface{})
		cuerpoinstrucciones.Parametros_instrucciones["Recurso"] = partesInstruccion[1]

	case "MUTEX_LOCK":
		cuerpoinstrucciones.Nombre_instruccion = "MUTEX_LOCK"
		cuerpoinstrucciones.Parametros_instrucciones = make(map[string]interface{})
		cuerpoinstrucciones.Parametros_instrucciones["Recurso"] = partesInstruccion[1]

	case "MUTEX_UNLOCK":
		cuerpoinstrucciones.Nombre_instruccion = "MUTEX_UNLOCK"
		cuerpoinstrucciones.Parametros_instrucciones = make(map[string]interface{})
		cuerpoinstrucciones.Parametros_instrucciones["Recurso"] = partesInstruccion[1]

	case "THREAD_EXIT":
		cuerpoinstrucciones.Nombre_instruccion = "THREAD_EXIT"
		cuerpoinstrucciones.Parametros_instrucciones = nil

	case "PROCESS_EXIT":
		cuerpoinstrucciones.Nombre_instruccion = "PROCESS_EXIT"
		cuerpoinstrucciones.Parametros_instrucciones = nil

	}

	respuesta, err := json.Marshal(BodyInstrucciones{
		Nombre_instruccion:       cuerpoinstrucciones.Nombre_instruccion,
		Parametros_instrucciones: cuerpoinstrucciones.Parametros_instrucciones,
	})
	//slog.Debug("Memoria Cpu - Se envio la instruccion", "Instruccion", instruccionAdevolver)

	formateada := FormatearInstruccion(cuerpoinstrucciones.Nombre_instruccion, cuerpoinstrucciones.Parametros_instrucciones)
	slog.Info(fmt.Sprintf("## Obtener instrucción - (PID:TID) - (%d:%d) - %s", pid, tid, formateada))

	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	time.Sleep(time.Duration(config.ResponseDelay) * time.Millisecond)
	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

func GuardarContextoEjecucion(w http.ResponseWriter, r *http.Request) { // El CPU termina, Memoria guarda
	//slog.Debug("Memoria Cpu - Se ingreso al pedido de guardar contexto de ejecucion")
	// El cpu deposita los registros desp de la ejec)
	var request BodyContextoEjecucion
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Aca se deberia actualizar el contexto de ejecucion en memoria
	pid := request.PID
	tid := request.TID
	registrosRecibidos := request.Registros

	var registrosAguardar RegistroCPU

	registrosAguardar.AX = registrosRecibidos.AX
	registrosAguardar.BX = registrosRecibidos.BX
	registrosAguardar.CX = registrosRecibidos.CX
	registrosAguardar.DX = registrosRecibidos.DX
	registrosAguardar.EX = registrosRecibidos.EX
	registrosAguardar.FX = registrosRecibidos.FX
	registrosAguardar.GX = registrosRecibidos.GX
	registrosAguardar.HX = registrosRecibidos.HX
	registrosAguardar.PC = registrosRecibidos.PC

	MapaTidContexto[PidYTid{PID: pid, TID: tid}] = registrosAguardar

	slog.Info(fmt.Sprintf("## Contexto Actualizado - (PID:TID) - (%d:%d)", pid, tid))

	respuesta := 1

	time.Sleep(time.Duration(config.ResponseDelay) * time.Millisecond)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(strconv.Itoa(respuesta)))
}

func DevolverContextoEjecucion(w http.ResponseWriter, r *http.Request) { // El CPU Pide, Memoria devuelve
	//slog.Debug("Memoria Cpu - Se ingreso al pedido de devolver contexto de ejecuccion")

	queryParams := r.URL.Query()
	pidRecibido := queryParams.Get("PID")
	tidRecibido := queryParams.Get("TID")
	pid, _ := strconv.Atoi(pidRecibido)
	tid, _ := strconv.Atoi(tidRecibido)

	registros := MapaTidContexto[PidYTid{PID: pid, TID: tid}]
	baseylimite := MapaPidContexto[pid]

	respuesta, err := json.Marshal(BodyRegistroCPU{
		AX:     registros.AX,
		BX:     registros.BX,
		CX:     registros.CX,
		DX:     registros.DX,
		EX:     registros.EX,
		FX:     registros.FX,
		GX:     registros.GX,
		HX:     registros.HX,
		PC:     registros.PC,
		Base:   baseylimite.Base,
		Limite: baseylimite.Limite,
	})

	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}
	slog.Info(fmt.Sprintf("## Contexto Solicitado - (PID:TID) - (%d:%d)", pid, tid))
	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

func AtenderEjecutarInstruccion(w http.ResponseWriter, r *http.Request) {
	//slog.Debug("Memoria Cpu - Se ingreso al pedido de ejecutar instruccion")

	queryParams := r.URL.Query()
	valorRecibido := queryParams.Get("VALOR")
	direccionRecibida := queryParams.Get("DIRECCIONFISICA")
	operacion := queryParams.Get("OPERACION")
	pid, _ := strconv.Atoi(queryParams.Get("PID"))
	tid, _ := strconv.Atoi(queryParams.Get("TID"))

	direccionInt, _ := strconv.Atoi(direccionRecibida)
	direccion := uint32(direccionInt)
	valorecibidoInt, _ := strconv.Atoi(valorRecibido)
	valorRecibidoUint32 := uint32(valorecibidoInt)

	switch operacion {
	case "WRITE":
		//slog.Debug("Memoria Cpu - Estancias del valor", "Original", valorRecibido, "Entero", valorecibidoInt, "Uint32", valorRecibidoUint32)
		// Escribe en memoria
		escribir := make([]byte, 4)
		binary.BigEndian.PutUint32(escribir, valorRecibidoUint32)
		//slog.Debug("Memoria Cpu - Se va a escribir en memoria", "Direccion", direccion, "Valor", fmt.Sprintf("%v", escribir))

		if len(escribir) != 4 {
			http.Error(w, "El valor debe tener exactamente 4 bytes", http.StatusBadRequest)
			return
		}
		copy(EspacioMemoria[direccion:direccion+4], escribir)

		slog.Info(fmt.Sprintf("## Escritura - (PID:TID) - (%d:%d) - Dir. Física: <%d> - Tamaño: <%d>", pid, tid, direccion, len(escribir)))
		//Si esta todo bien, se devuelve un OK
		respuesta := []byte("OK")
		w.WriteHeader(http.StatusOK)
		w.Write(respuesta)
	case "READ":
		// Lee el valor a memoria y devuelve el valor leido
		leido := EspacioMemoria[direccion : direccion+4]
		slog.Info(fmt.Sprintf("## Lectura - (PID:TID) - (%d:%d) - Dir. Física: <%d> - Tamaño: <%d>", pid, tid, direccion, len(leido)))
		w.WriteHeader(http.StatusOK)
		w.Write(leido)
	}
}

/* ------------------------------AUX------------------------------------ */
func FormatearInstruccion(nombre string, parametros map[string]interface{}) string {
	var partes []string
	partes = append(partes, fmt.Sprintf("Instrucción: %s", nombre))

	if parametros != nil {
		for clave, valor := range parametros {
			partes = append(partes, fmt.Sprintf("%s: %v", clave, valor))
		}
	}

	return strings.Join(partes, ", ")
}
