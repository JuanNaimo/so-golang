package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
)

/* --------------- Fetch Instruccion --------------- */
func fetchInstruccion(pid int, tid int) BodyInstrucciones {
	var instruccion BodyInstrucciones

	cliente := &http.Client{}
	url := fmt.Sprintf("http://%s:%d/memoria/pedido_instrucciones", config.IPMemory, config.PortMemory)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		slog.Error("Error")
	}
	q := req.URL.Query()
	q.Add("PID", strconv.Itoa(pid))
	q.Add("TID", strconv.Itoa(tid))
	q.Add("PC", strconv.Itoa(int(registrosCPU.PC)))
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

	err = json.Unmarshal(bodyBytes, &instruccion)
	if err != nil {
		slog.Error("Error")
	}
	slog.Info(fmt.Sprintf("## TID: %d - FETCH - Program Counter: %d", tid, registrosCPU.PC))

	slog.Debug("Fetch - ", "Instruccion Nombre", instruccion.Nombre_instruccion, "Instruccion Parametros", instruccion.Parametros_instrucciones, "PC", registrosCPU.PC)
	return instruccion
}

/* --------------- Decode Instruccion --------------- */
func decodeInstruccion(instruccion BodyInstrucciones) BodyInstrucciones {
	// Calcular MMU para las instrucciones que requieren memoria logica (WRITE_MEM, READ_MEM)
	if instruccion.Nombre_instruccion == "WRITE_MEM" || instruccion.Nombre_instruccion == "READ_MEM" {
		direccionLogica := obtenerValorRegistro(instruccion.Parametros_instrucciones["RegistroDireccion"].(string))
		slog.Debug("El registro de direccion es: ", "Registro", instruccion.Parametros_instrucciones["RegistroDireccion"].(string))
		slog.Debug("Instrucciones - Decode - ", "Direccion Logica", direccionLogica)

		direccionFisica, err := CalcularMMU(direccionLogica, obtenerValorRegistro("Base"), obtenerValorRegistro("Limite"))
		if err != nil {
			slog.Debug("Error Instrucciones - ", "Error", err)
			seguirCicloInstrucciones = false
			SetMotivoDesalojo("Segmentation Fault")
		}
		instruccion.Parametros_instrucciones["DireccionFisica"] = direccionFisica

	}

	return instruccion
}

/* --------------- Ejecutar Instruccion --------------- */
func ejecutarInstruccion(instruccionInterpretada BodyInstrucciones, tid int) {

	slog.Info(fmt.Sprintf("## TID: %d - Ejecutando: %s - %s", tid, instruccionInterpretada.Nombre_instruccion, formatearParametros(instruccionInterpretada.Parametros_instrucciones)))
	switch instruccionInterpretada.Nombre_instruccion {
	case "SET":

		valorInt, _ := strconv.Atoi(instruccionInterpretada.Parametros_instrucciones["Valor"].(string))

		valorUint32 := uint32(valorInt)

		ejecutarSet(instruccionInterpretada.Parametros_instrucciones["Registro"].(string), valorUint32)
	case "READ_MEM":
		slog.Info(fmt.Sprintf("## TID: %d - Acción: LEER - Dirección Física: %d", tid, instruccionInterpretada.Parametros_instrucciones["DireccionFisica"].(uint32)))

		ejecutarREAD_MEM(instruccionInterpretada.Parametros_instrucciones["RegistroDatos"].(string), instruccionInterpretada.Parametros_instrucciones["DireccionFisica"].(uint32))
	case "WRITE_MEM":
		slog.Info(fmt.Sprintf("## TID: %d - Acción: ESCRIBIR - Dirección Física: %d", tid, instruccionInterpretada.Parametros_instrucciones["DireccionFisica"].(uint32)))

		ejecutarWRITE_MEM(instruccionInterpretada.Parametros_instrucciones["DireccionFisica"].(uint32), instruccionInterpretada.Parametros_instrucciones["RegistroDatos"].(string))
	case "SUM":
		ejecutarSUM(instruccionInterpretada.Parametros_instrucciones["RegistroDestino"].(string), instruccionInterpretada.Parametros_instrucciones["RegistroOrigen"].(string))
	case "SUB":
		ejecutarSUB(instruccionInterpretada.Parametros_instrucciones["RegistroDestino"].(string), instruccionInterpretada.Parametros_instrucciones["RegistroOrigen"].(string))
	case "JNZ":
		instruccionInt, _ := strconv.Atoi(instruccionInterpretada.Parametros_instrucciones["Instruccion"].(string))
		valorUint32 := uint32(instruccionInt)

		ejecutarJNZ(instruccionInterpretada.Parametros_instrucciones["Registro"].(string), valorUint32)
	case "LOG":
		ejecutarLOG(instruccionInterpretada.Parametros_instrucciones["Registro"].(string))
	case "DUMP_MEMORY":
		ejecutarDUMP_MEMORY()
	case "IO":
		ejecutarIO(instruccionInterpretada.Parametros_instrucciones["Tiempo"].(string))

	case "PROCESS_CREATE":

		ejecutarPROCESS_CREATE(instruccionInterpretada.Parametros_instrucciones["ArchivoInstrucciones"].(string), instruccionInterpretada.Parametros_instrucciones["Tamanio"].(string), instruccionInterpretada.Parametros_instrucciones["Prioridad"].(string))

	case "THREAD_CREATE":
		ejecutarTHREAD_CREATE(instruccionInterpretada.Parametros_instrucciones["ArchivoInstrucciones"].(string), instruccionInterpretada.Parametros_instrucciones["Prioridad"].(string))
	case "THREAD_JOIN":

		ejecutarTHREAD_JOIN(instruccionInterpretada.Parametros_instrucciones["Tid"].(string))

	case "THREAD_CANCEL":
		ejecutarTHREAD_CANCEL(instruccionInterpretada.Parametros_instrucciones["Tid"].(string))

	case "MUTEX_CREATE":
		ejecutarMUTEX_CREATE(instruccionInterpretada.Parametros_instrucciones["Recurso"].(string))

	case "MUTEX_LOCK":
		ejecutarMUTEX_LOCK(instruccionInterpretada.Parametros_instrucciones["Recurso"].(string))

	case "MUTEX_UNLOCK":
		ejecutarMUTEX_UNLOCK(instruccionInterpretada.Parametros_instrucciones["Recurso"].(string))

	case "THREAD_EXIT":
		ejecutarTHREAD_EXIT()

	case "PROCESS_EXIT":
		ejecutarPROCESS_EXIT()

	default:
		slog.Error("Instrucciones - Error al ejecutar la instruccion, no se encontro la instruccion:", "Instruccion", instruccionInterpretada.Nombre_instruccion)
	}
}

func ejecutarSet(registro string, valor uint32) {
	//Setea el valor en el registro
	slog.Debug("Valor a setear en el registro", "Registro", registro, "Valor", valor)

	if registro == "AX" {
		registrosCPU.AX = valor
	}
	if registro == "BX" {
		registrosCPU.BX = valor
	}
	if registro == "CX" {
		registrosCPU.CX = valor
	}
	if registro == "DX" {
		registrosCPU.DX = valor
	}
	if registro == "EX" {
		registrosCPU.EX = valor
	}
	if registro == "FX" {
		registrosCPU.FX = valor
	}
	if registro == "GX" {
		registrosCPU.GX = valor
	}
	if registro == "HX" {
		registrosCPU.HX = valor
	}

}

func ejecutarSUM(registro_destino string, registro_origen string) {
	//Suma el valor de un registro origen al registro destino
	valor_origen := obtenerValorRegistro(registro_origen)
	valor_destino := obtenerValorRegistro(registro_destino)
	suma := valor_origen + valor_destino
	ejecutarSet(registro_destino, suma)
}

func ejecutarSUB(registro_destino string, registro_origen string) {
	//Resta el valor de un registro origen al registro destino
	valor_origen := obtenerValorRegistro(registro_origen)
	valor_destino := obtenerValorRegistro(registro_destino)
	slog.Debug("Valores a usar en SUB: ", "Valor origen", valor_origen, "Valor destino", valor_destino)
	resta := valor_destino - valor_origen
	ejecutarSet(registro_destino, resta)
}

func ejecutarJNZ(registro string, num_instruccion_a_saltar uint32) {
	//Salta a la instruccion indicada si el registro es distinto de 0
	valor_registro := obtenerValorRegistro(registro)
	slog.Debug("El valor del registro es: ", "Valor", valor_registro)
	if valor_registro != 0 {
		registrosCPU.PC = num_instruccion_a_saltar - 1 // Se resta 1 porque en el ciclo de instrucciones se incrementa el PC
	}
}
func ejecutarLOG(registro string) {
	//Loguea el valor de un registro
	valor_registro := obtenerValorRegistro(registro)

	slog.Info(fmt.Sprintf("## El valor del registro %s es %d", registro, valor_registro))
}

func ejecutarREAD_MEM(registroDatos string, direccionFisica uint32) {

	valor := ejecutarInstruccionMemoria(0, direccionFisica, "READ")

	ejecutarSet(registroDatos, valor)
}

func ejecutarWRITE_MEM(direccionFisica uint32, registroDatos string) {

	valor := obtenerValorRegistro(registroDatos)

	ejecutarInstruccionMemoria(valor, direccionFisica, "WRITE")
}

// ESTAS INSTRUCCIONES NO SE EJECUTAN, SE ENVIAN A KERNEL COMO SYSCALL Y ESTE SE ENCARGA DE EJECUTARLAS
func ejecutarDUMP_MEMORY() {
	EnviarSyscall("DUMP_MEMORY", nil)
	seguirCicloInstrucciones = false
	SetMotivoDesalojo("SYSCALL")
}

func ejecutarIO(tiempo string) {
	EnviarSyscall("IO", map[string]interface{}{"tiempo": tiempo})
	seguirCicloInstrucciones = false
	SetMotivoDesalojo("SYSCALL")
}

func ejecutarPROCESS_CREATE(archivoInstrucciones string, tamanio string, prioridad string) {
	EnviarSyscall("PROCESS_CREATE", map[string]interface{}{"archivoInstrucciones": archivoInstrucciones, "tamanio": tamanio, "prioridad": prioridad})
	// Probando ahora
	seguirCicloInstrucciones = false
	SetMotivoDesalojo("SYSCALL")
}

func ejecutarTHREAD_CREATE(archivoInstrucciones string, prioridadString string) {
	EnviarSyscall("THREAD_CREATE", map[string]interface{}{"archivoInstrucciones": archivoInstrucciones, "prioridad": prioridadString})
	slog.Debug("THREAD_CREATE - Se envio la syscall")
	// Probando ahora
	seguirCicloInstrucciones = false
	SetMotivoDesalojo("SYSCALL")
}

func ejecutarTHREAD_JOIN(tidString string) {
	EnviarSyscall("THREAD_JOIN", map[string]interface{}{"tid": tidString})
	// Probando ahora
	seguirCicloInstrucciones = false
	SetMotivoDesalojo("SYSCALL")
}

func ejecutarTHREAD_CANCEL(tid string) {
	EnviarSyscall("THREAD_CANCEL", map[string]interface{}{"tid": tid})
	// Probando ahora
	seguirCicloInstrucciones = false
	SetMotivoDesalojo("SYSCALL")
}

func ejecutarMUTEX_CREATE(recurso string) {
	EnviarSyscall("MUTEX_CREATE", map[string]interface{}{"recurso": recurso})
	// Probando ahora
	seguirCicloInstrucciones = false
	SetMotivoDesalojo("SYSCALL")
}

func ejecutarMUTEX_LOCK(recurso string) {
	EnviarSyscall("MUTEX_LOCK", map[string]interface{}{"recurso": recurso})
	// Probando ahora
	seguirCicloInstrucciones = false
	SetMotivoDesalojo("SYSCALL")
}

func ejecutarMUTEX_UNLOCK(recurso string) {
	EnviarSyscall("MUTEX_UNLOCK", map[string]interface{}{"recurso": recurso})
	// Probando ahora
	seguirCicloInstrucciones = false
	SetMotivoDesalojo("SYSCALL")
}

func ejecutarTHREAD_EXIT() {
	EnviarSyscall("THREAD_EXIT", nil)
	seguirCicloInstrucciones = false
	SetMotivoDesalojo("PIOLA")
}

func ejecutarPROCESS_EXIT() {
	EnviarSyscall("PROCESS_EXIT", nil)
	seguirCicloInstrucciones = false
	SetMotivoDesalojo("SYSCALL")
}

func DetenerEjecucion() {
	atomic.StoreInt32(&HayInterrupcion, 0)
}

func obtenerValorRegistro(registro string) uint32 {
	//Obtiene el valor de un registro
	switch registro {
	case "AX":
		return registrosCPU.AX
	case "BX":
		return registrosCPU.BX
	case "CX":
		return registrosCPU.CX
	case "DX":
		return registrosCPU.DX
	case "EX":
		return registrosCPU.EX
	case "FX":
		return registrosCPU.FX
	case "GX":
		return registrosCPU.GX
	case "HX":
		return registrosCPU.HX
	case "Base":
		return registrosCPU.Base
	case "Limite":
		return registrosCPU.Limite
	default:
		return 0
	}
}

func ejecutarInstruccionMemoria(valor uint32, direccionFisica uint32, operacion string) uint32 {
	cliente := &http.Client{}
	url := fmt.Sprintf("http://%s:%d/memoria/ejecutar_instruccion", config.IPMemory, config.PortMemory)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		slog.Error("Error")
	}
	q := req.URL.Query()
	q.Add("VALOR", strconv.Itoa(int(valor)))
	q.Add("DIRECCIONFISICA", strconv.Itoa(int(direccionFisica)))
	q.Add("OPERACION", operacion)
	q.Add("PID", strconv.Itoa(pidEjecucion))
	q.Add("TID", strconv.Itoa(tidEjecucion))
	req.URL.RawQuery = q.Encode()
	slog.Debug("---------------------------> Se ejecuto la instruccion de memoria", "Valor", valor, "DireccionFisica", direccionFisica, "Operacion", operacion)
	req.Header.Set("Content-Type", "application/json")
	respuesta, err := cliente.Do(req)
	if err != nil {
		slog.Error("Error")
	}
	// Verificar el código de estado de la respuesta
	if respuesta.StatusCode != http.StatusOK {
		slog.Error("Error")
	}

	body, err := io.ReadAll(respuesta.Body)
	if err != nil {
		slog.Error("Error al leer el cuerpo de la respuesta:", "Error", err)
		return 0
	}

	// Convertir el cuerpo de la respuesta en un uint32
	if string(body) == "OK" {
		//Si esta todo bien, memoria devuelve un OK que se funco joya la escritura desde el otro lado
		return 0
	} else {
		valorLeido := binary.BigEndian.Uint32(body)
		valorLeidoUint32 := uint32(valorLeido)
		return valorLeidoUint32
	}

}
func SetMotivoDesalojo(motivo string) {
	MotivoDesalojo = motivo
}

/*----------------------*/
func formatearParametros(parametros map[string]interface{}) string {
	var partes []string
	for clave, valor := range parametros {
		partes = append(partes, fmt.Sprintf("%s: %v", clave, valor))
	}
	return strings.Join(partes, ", ")
}
