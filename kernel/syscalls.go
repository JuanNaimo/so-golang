package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/sisoputnfrba/tp-golang/utils"
)

func AtenderSyscalls(w http.ResponseWriter, r *http.Request) {
	var request SyscallBodyRequest
	//lo capturado en el Body, queda almacenado en request
	//el newdecoder lee lo que llega en el el body y el decode lo almacena siguiendo la estructura creada que le asignamos a la variable request

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	slog.Info(fmt.Sprintf("## (%d:%d) - Solicito syscall: %s", pidEjecutando, tidEjecutando, request.Nombre_syscall))

	switch request.Nombre_syscall {
	case "DUMP_MEMORY":
		sysDUMP_MEMORY()
	case "PROCESS_CREATE":

		mutexReady.Lock()
		hiloAProcesar.Estado = Ready
		ColaReady.Enqueue(hiloAProcesar) // Reinsertar el hilo en la cola de Ready
		colaReadyCond.Signal()           // Notifica que se añadió un hilo a la cola de ready
		mutexReady.Unlock()

		archivo := request.Parametros_syscall["archivoInstrucciones"].(string)

		tamanioString := request.Parametros_syscall["tamanio"].(string)
		tamanio, _ := strconv.Atoi(tamanioString)

		prioridadString := request.Parametros_syscall["prioridad"].(string)
		prioridad, _ := strconv.Atoi(prioridadString)
		slog.Debug("Log del sysProcess create", "Archivo", archivo, "Tamanio", tamanio, "Prioridad", prioridad)
		sysPROCESS_CREATE(archivo, tamanio, prioridad)
	case "PROCESS_EXIT":
		sysPROCCESS_EXIT()
	case "THREAD_CREATE":

		mutexReady.Lock()
		hiloAProcesar.Estado = Ready
		ColaReady.Enqueue(hiloAProcesar) // Reinsertar el hilo en la cola de Ready
		colaReadyCond.Signal()           // Notifica que se añadió un hilo a la cola de ready
		mutexReady.Unlock()

		archivo := request.Parametros_syscall["archivoInstrucciones"].(string)

		//path := pathCarpetaPrueba + archivo
		path, _ := filepath.Abs(filepath.Join("./prueba/") + "/" + archivo)
		slog.Debug("Path final: ", "Path", path)

		prioridadString := request.Parametros_syscall["prioridad"].(string)
		prioridad, _ := strconv.Atoi(prioridadString)

		sysTHREAD_CREATE(path, prioridad)
	case "THREAD_JOIN":

		tidString := request.Parametros_syscall["tid"].(string)
		tid, _ := strconv.Atoi(tidString)

		sysTHREAD_JOIN(tid)
	case "THREAD_CANCEL":
		tidString := request.Parametros_syscall["tid"].(string)
		tid, _ := strconv.Atoi(tidString)

		mutexReady.Lock()
		hiloAProcesar.Estado = Ready
		ColaReady.Enqueue(hiloAProcesar) // Reinsertar el hilo en la cola de Ready
		colaReadyCond.Signal()           // Notifica que se añadió un hilo a la cola de ready
		mutexReady.Unlock()

		sysTHREAD_CANCEL(tid)
	case "THREAD_EXIT":
		tid := tidEjecutando
		sysTHREAD_EXIT(tid)
	case "MUTEX_LOCK":
		recurso := request.Parametros_syscall["recurso"].(string)
		sysMUTEX_LOCK(recurso)
	case "MUTEX_UNLOCK":
		recurso := request.Parametros_syscall["recurso"].(string)
		sysMUTEX_UNLOCK(recurso)
	case "MUTEX_CREATE":

		mutexReady.Lock()
		hiloAProcesar.Estado = Ready
		ColaReady.Enqueue(hiloAProcesar) // Reinsertar el hilo en la cola de Ready
		colaReadyCond.Signal()           // Notifica que se añadió un hilo a la cola de ready
		mutexReady.Unlock()

		recurso := request.Parametros_syscall["recurso"].(string)
		sysMUTEX_CREATE(recurso)
	case "IO":
		tiempoString := request.Parametros_syscall["tiempo"].(string)
		tiempo, _ := strconv.Atoi(tiempoString)

		sysIO(tiempo)
	}
}

func sysPROCESS_CREATE(archivoRuta string, tamanioMemoria int, prioridad int) {

	//Crear un nuevo proceso
	CrearProceso(archivoRuta, tamanioMemoria, prioridad)
}

func sysPROCCESS_EXIT() {
	//Chekear si el hilo que ejecuto es el 0
	if tidEjecutando == 0 {

		FinalizarProceso(&pcbEjecutando)
	} else {
		slog.Debug("ERROR. Solo el hilo 0 puede finalizar el proceso")
	}
}

func sysDUMP_MEMORY() {
	//Se bloquea el hilo hasta que se reciba la respuesta
	tcbEjecutando.Estado = Blocked

	// Enviar a memoria
	respuesta := enviarDumpMemory(tidEjecutando, pidEjecutando)

	switch respuesta {
	case 1: //Dump exitoso, se pasa el hilo a ready
		slog.Debug("Se realizo el dump de memoria correctamente")
		tcbEjecutando.Estado = Ready
		mutexReady.Lock()
		ColaReady.Enqueue(tcbEjecutando) //DUDA: No habria que pasarle un puntero a tcbEjecutando? En otro lado me surgio la misma duda. Sino, hay que iniciar un tcbNuevo y pasarle los valores de tcbEjecutando
		colaReadyCond.Signal()
		mutexReady.Unlock()
	case 0: //Dump fallido, se pasa el proceso a exit
		slog.Debug("Error al realizar el dump de memoria. Finalizando proceso")
		//Luego finalizar el proceso
		FinalizarProceso(&pcbEjecutando)

		slog.Debug("Error al realizar el dump de memoria. Finalizando proceso")

	default:
		slog.Debug("Error desconocido", "Respuesta", respuesta)
	}
}

func sysTHREAD_CREATE(archivoRuta string, prioridad int) {
	punteroHilo := punteroPCBporTCB(tcbEjecutando)
	CrearHilo(punteroHilo, prioridad, archivoRuta)
}
func sysTHREAD_JOIN(tid int) {
	//Si existe el tid, pasarlo a blockeados
	if existeTID(tid, pcbEjecutando.TIDs) {
		MapaBlockedJoin[tid] = tcbEjecutando

		slog.Info(fmt.Sprintf("## (%d:%d)-Bloqueado por: THREAD_JOIN", pcbEjecutando.PID, tcbEjecutando.TID))
	} else { //Si no existe, ignorar
		slog.Debug("No existe el tid", "TID", tid)

		mutexReady.Lock()
		hiloAProcesar.Estado = Ready
		ColaReady.Enqueue(hiloAProcesar) // Reinsertar el hilo en la cola de Ready
		colaReadyCond.Signal()           // Notifica que se añadió un hilo a la cola de ready
		mutexReady.Unlock()
	}
}

func sysTHREAD_CANCEL(tid int) {
	//Si existe el tid, finalizarlo
	if existeTID(tid, pcbEjecutando.TIDs) {

		// punteroDelHilo := punteroPCBporTCB(tid)
		mutexReady.Lock()

		// Crear una cola auxiliar para almacenar los hilos que no serán eliminados
		auxColaReady := utils.Cola[TCB]{}

		// Iterar sobre la ColaReady y eliminar el hilo con el TID especificado
		for !ColaReady.EstaVacia() {
			hilo := ColaReady.Dequeue() // Extrae un hilo de la cola
			if hilo.TID != tid {
				auxColaReady.Enqueue(hilo) // Mantén los demás hilos
			}
		}

		// Actualizar la ColaReady con los elementos restantes
		ColaReady.Elementos = auxColaReady.Elementos
		mutexReady.Unlock()

		FinalizarHilo(&pcbEjecutando, tid) // Un hilo solo puede finalizar hilos del mismo proceso

	}
	//Si no existe, ignorar
}
func sysTHREAD_EXIT(tid int) {
	// Pedir a memoria que finalize el hilo
	slog.Debug("Finalizando hilo", "Hilo", tid)

	// Cambiar el estado del hilo a exit
	tcbEjecutando.Estado = Exit

	ColaExit.Enqueue(tcbEjecutando)
	colaExitCond.Signal() // Notifica que se añadió un hilo a la cola de exit

	//FinalizarHilo(&pcbEjecutando, tid)
}

func sysMUTEX_LOCK(recurso string) {
	proceso := punteroPCBporTCB(tcbEjecutando)
	if proceso == nil {
		slog.Error("Error: Proceso no encontrado para el TCB ejecutando", "TCB", tcbEjecutando)
		return
	}

	if recursoEnMutex(proceso, recurso) {
		if !recursoTomado(proceso, recurso) {

			mutex := obtenerMutexPorRecurso(proceso, recurso)

			mutex.asignado = true
			//mutex.hiloAsignado = &tcbEjecutando
			mutex.TIDhiloasignado = tcbEjecutando.TID

			slog.Debug("Mutex asignado a", "PID", proceso.PID, "Hilo", tcbEjecutando.TID, "Nombre Mutex", mutex)

			mutexReady.Lock()
			hiloAProcesar.Estado = Ready
			ColaReady.Enqueue(hiloAProcesar) // Reinsertar el hilo en la cola de Ready
			colaReadyCond.Signal()           // Notifica que se añadió un hilo a la cola de ready
			mutexReady.Unlock()

		} else {
			mutex := obtenerMutexPorRecurso(proceso, recurso)
			tcbEjecutando.Estado = Blocked
			mutex.colaBloqueados.Enqueue(tcbEjecutando)
			slog.Info(fmt.Sprintf("## (%d:%d)-Bloqueado por: MUTEX", pcbEjecutando.PID, tcbEjecutando.TID))
			slog.Debug("Cola de bloqueados por mutex", "PID", proceso.PID, "Cola de bloqueados", mutex.colaBloqueados)
		}
	}

}

func sysMUTEX_UNLOCK(recurso string) {
	hiloAReady := tcbEjecutando
	mutexReady.Lock()
	hiloAReady.Estado = Ready
	ColaReady.Enqueue(hiloAReady) // Reinsertar el hilo en la cola de Ready
	colaReadyCond.Signal()        // Notifica que se añadió un hilo a la cola de ready
	mutexReady.Unlock()

	proceso := punteroPCBporTCB(tcbEjecutando)

	if recursoEnMutex(proceso, recurso) {
		mutex := obtenerMutexPorRecurso(proceso, recurso)
		slog.Debug("Entro al if de recurso en mutex dentro de mutex ubnlock")

		if mutex.TIDhiloasignado == tcbEjecutando.TID {
			hilodesbloqueado := mutex.colaBloqueados.Dequeue()
			slog.Debug("Hilo desbloqueado", "Hilo", hilodesbloqueado)
			hilodesbloqueado.Estado = Ready
			mutex.TIDhiloasignado = hilodesbloqueado.TID
			//mutex.hiloAsignado = &hilodesbloqueado
			slog.Debug("Mutex asginado al hilo", "Hilo", mutex.TIDhiloasignado)
			ColaReady.Enqueue(hilodesbloqueado)
			colaReadyCond.Signal()
			slog.Debug("Se asigno el mutex al hilo y se lo agrego a la cola de reay", "Hilo", hilodesbloqueado, "Estado de la cola de Bloqeuados", mutex.colaBloqueados)
		}
	}

}

func sysMUTEX_CREATE(recurso string) {
	mutex := new(MUTEX)

	mutex.recurso = recurso
	mutex.asignado = false
	mutex.TIDhiloasignado = -1 //Osea no lo tiene ningun hilo
	//	mutex.hiloAsignado = nil
	mutex.colaBloqueados = &utils.Cola[TCB]{}

	slog.Debug("Mutex creado", "Mutex", mutex)
	proceso := punteroPCBporTCB(tcbEjecutando)
	proceso.Mutex = append(proceso.Mutex, *mutex)

}

func sysIO(tiempo int) {
	slog.Debug("Se entro a ejecutar la SYSCALL de IO", "Tiempo", tiempo)
	mutexReady.Lock()

	hiloBloqueado := tcbEjecutando

	mutexReady.Unlock()

	//Tiempo que se va a quedar haciendo IO
	hiloBloqueado.Estado = Blocked
	//Agregar a bloqueados, no si a la cola porque no se va a ejecutar hasta que termine el IO y despues hay que retirarlo para pasarlo a ready. Hay que ver como hacer
	slog.Info(fmt.Sprintf("## (%d:%d)-Bloqueado por: IO", hiloBloqueado.PID, hiloBloqueado.TID))
	go func() {
		time.Sleep(time.Duration(tiempo))
		//Desbloquear hilo
		hiloBloqueado.Estado = Ready
		mutexReady.Lock()
		ColaReady.Enqueue(hiloBloqueado)
		colaReadyCond.Signal()
		mutexReady.Unlock()
	}()
	slog.Info(fmt.Sprintf("## (%d:%d)- Finalizo IO y pasa a READY", hiloBloqueado.PID, hiloBloqueado.TID))
}

func recursoEnMutex(pcb *PCB, recurso string) bool {
	slog.Debug("Recurso a buscar", "Recurso", recurso)
	for i := range pcb.Mutex {
		slog.Debug("Recurso encontrado en mutex", "Recurso", pcb.Mutex[i].recurso)
		if pcb.Mutex[i].recurso == recurso {
			return true
		}
	}
	slog.Debug("Recurso no encontrado en mutex", "Recurso", recurso)
	return false
}

func recursoTomado(pcb *PCB, recurso string) bool {
	for i := range pcb.Mutex {
		if pcb.Mutex[i].recurso == recurso {
			slog.Debug("Recurso tomado", "Recurso", recurso)
			return pcb.Mutex[i].asignado
		}
	}
	slog.Debug("Recurso no esta tomado", "Recurso", recurso)
	return false
}

func obtenerMutexPorRecurso(pcb *PCB, recurso string) *MUTEX {
	for i := range pcb.Mutex {
		if pcb.Mutex[i].recurso == recurso {
			slog.Debug("Mutex encontrado", "Recurso", recurso)
			return &pcb.Mutex[i]
		}
	}
	slog.Debug("No se encontro el mutex", "Recurso", recurso)
	return nil
}

func enviarDumpMemory(tid int, pid int) int {
	cliente := &http.Client{}
	url := fmt.Sprintf("http://%s:%d/memoria/dumpmemory", config.IPMemory, config.PortMemory)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return -1
	}

	q := req.URL.Query()
	q.Add("PID", strconv.Itoa(pid))
	q.Add("TID", strconv.Itoa(tid))
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Content-Type", "application/json")
	respuesta, err := cliente.Do(req)
	if err != nil {
		return -1
	}

	// Verificar el código de estado de la respuesta
	if respuesta.StatusCode != http.StatusOK {
		return -1
	}

	bodyBytes, err := io.ReadAll(respuesta.Body)
	if err != nil {
		return -1
	}
	//Estado 1: Dump exitoso
	//Estado 0: Dump fallido
	var respuestaDump int
	_ = json.Unmarshal(bodyBytes, &respuestaDump)

	slog.Debug("Respuesta de memoria", "Respuesta", respuestaDump)
	return respuestaDump
}

func existeTID(tid int, tids []int) bool {
	for _, t := range tids {
		if t == tid {
			return true
		}
	}
	return false
}
