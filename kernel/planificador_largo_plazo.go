package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/sisoputnfrba/tp-golang/utils"
)

func PlanificadorLargoPlazo() {
	// Iniciar hilo de proceso de cola new
	go procesarColaNew()

	// Iniciar hilo de proceso de cola exit
	go procesarColaExit()

	select {}
}

/* ---------------- Creación de Procesos ---------------- */
var colaNewCond = sync.NewCond(&mutexNew)
var memCond = sync.NewCond(&mutexMemoria)

func procesarColaNew() {
	for {
		// Semaforo New
		mutexNew.Lock()
		for ColaNew.EstaVacia() {
			colaNewCond.Wait() // Espera hasta que haya un proceso en ColaNew
		}

		// Se saca el proceso de New
		pcbAcrear := ColaNew.Dequeue()
		mutexNew.Unlock()

		// Se hace el pedido a memoria
		estadoMemoria := 0
		//Estado 1: Creacion exitosa del proceso en memoria
		//Estado 0: Creacion fallida del proceso en memoria
		for estadoMemoria != 1 {
			estadoMemoria = PeticionCrearProcesoAMemoria(pcbAcrear.PID, pcbAcrear.Tamanio)
			slog.Debug("Estado de la creacion del proceso en memoria: ", "PID", pcbAcrear.PID, "Estado", estadoMemoria)
		}

		slog.Debug("Largo Plazo - Se creo el proceso -", "PID", pcbAcrear.PID)

		if estadoMemoria == 1 {
			// Se crea el hilo 0 del proceso - No necesita sem new porque lo tiene adentro
			CrearHilo(&pcbAcrear, pcbAcrear.PrioridadHilo)

			// Semaforo Ready
			mutexProcesos.Lock()

			// Se envia el proceso a colaProcesos
			ColaProcesos.Enqueue(pcbAcrear)
			colaProcesosCond.Signal()
			mutexProcesos.Unlock()
		}
	}
}

// Funcion para crear procesos que recibe el nombre del archivo del seudocodigo, el tamaño del proceso en memoria
func CrearProceso(archivo string, tamanioMemoria int, prioridadHilo int) {
	pidContador++
	slog.Debug("Nombre del archivo", "Archivo", archivo)
	ruta, _ := filepath.Abs(filepath.Join("./prueba/") + "/" + archivo)

	slog.Debug("Ruta sumandole el archivo: ", "Ruta", ruta)
	// Crear Proceso
	proceso := PCB{
		PID:           pidContador,
		TIDs:          []int{},   // El slice se inicializa en cero para que despues sea autoincremental a partir de eso
		Mutex:         []MUTEX{}, // Se inicializa un slice vacio de mutex, pa' despues agregarle
		Tamanio:       tamanioMemoria,
		Estado:        New,
		Archivo:       ruta,
		PrioridadHilo: prioridadHilo,
		ContadorTID:   0,
	}

	mutexNew.Lock()

	ColaNew.Enqueue(proceso)
	colaNewCond.Signal() // Notifica que se añadió un proceso a la cola
	mutexNew.Unlock()

	slog.Info(fmt.Sprintf("## (%d:0) Se crea el proceso - Estado: NEW", proceso.PID))
}

// Funcion para crear hilos, recibe al proceso padre y el nivel de prioridad
func CrearHilo(proceso *PCB, prioridad int, archivo ...string) {
	hilo := TCB{
		Prioridad: prioridad,
		Quantum:   config.Quantum,
		PID:       proceso.PID,
		Estado:    Ready,
	}
	//slog.Debug("Direccion para ver si es el mismo proceso", "Proceso", *proceso)

	hilo.TID = proceso.ContadorTID
	proceso.ContadorTID++

	proceso.TIDs = append(proceso.TIDs, hilo.TID)

	// slog.Debug(("Se agregó el hilo al array de tids del proceso"), "TID", hilo.TID, "PID", hilo.PID, "Tamanio", len(proceso.TIDs))
	// slog.Debug("Estado del array de tids del proceso", "TIDs", proceso.TIDs)

	if len(archivo) != 0 {

		slog.Debug("Ruta: ", "Ruta", archivo[0])

		err := PeticionCrearHiloAMemoria(proceso.PID, hilo.TID, archivo[0])

		if err == -1 {
			slog.Error("Largo Plazo - Error al crear el hilo en peticion a memoria con archivo externo -", "TID", hilo.TID)
		}
	} else {
		err := PeticionCrearHiloAMemoria(proceso.PID, hilo.TID, proceso.Archivo)

		if err == -1 {
			slog.Error("Largo Plazo - Error al crear el hilo en peticion a memoria con archivo de proceso -", "TID", hilo.TID)
		}
	}

	mutexReady.Lock()

	ColaReady.Enqueue(hilo)
	colaReadyCond.Signal() // Notifica que se añadió un hilo a la cola

	mutexReady.Unlock()

	// slog.Debug("Largo Plazo - Se creo el hilo -", "TID", hilo.TID)
	slog.Info(fmt.Sprintf("## (%d:%d) Se crea el Hilo - Estado: READY", proceso.PID, hilo.TID))
}

func PeticionCrearHiloAMemoria(pid int, tid int, archivoRuta string) int {

	body, err := json.Marshal(RequestPeticionHilo{
		PID:              pid,
		TID:              tid,
		RutaPseudocodigo: archivoRuta,
	})
	if err != nil {
		return -1
	}

	cliente := &http.Client{}
	url := fmt.Sprintf("http://%s:%d/memoria/crearhilo", config.IPMemory, config.PortMemory)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return -1
	}

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

	est := string(bodyBytes)
	estadoCreacion, _ := strconv.Atoi(est)

	slog.Debug("Se envio la peticion a memoria a crear hilo")
	slog.Debug("Estado de la creacion del hilo: ", "Estado", estadoCreacion)
	return estadoCreacion
}

type RequestPeticion struct {
	PID              int    `json:"PID"`
	TamanioProceso   int    `json:"TamanioProceso"`
	RutaPseudocodigo string `json:"RutaPseudocodigo"`
}

func PeticionCrearProcesoAMemoria(pid int, tamanioProceso int) int {

	body, err := json.Marshal(RequestPeticion{
		PID:            pid,
		TamanioProceso: tamanioProceso,
	})
	if err != nil {
		return -1
	}

	cliente := &http.Client{}
	url := fmt.Sprintf("http://%s:%d/memoria/crearproceso/{pid}", config.IPMemory, config.PortMemory)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return -1
	}

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

	est := string(bodyBytes)
	estadoCreacion, _ := strconv.Atoi(est)

	// slog.Debug("Antes del mutex de memoria")

	switch estadoCreacion {
	case 0:
		// Mutex Condicional de cuando se libera un proceso
		mutexMemoria.Lock()
		memCond.Wait()
		mutexMemoria.Unlock()
		//slog.Debug("------------ eeeeeeeeeeeeeeeeeeee coso")
		return 0
	case 1:
		return 1
	default:
		slog.Error("Largo Plazo - Error al comprobar peticion para crear a memoria")
		return -1
	}
}

/* ---------------- Finalización de Procesos ---------------- */
var colaExitCond = sync.NewCond(&mutexExit)

func procesarColaExit() {
	for {
		// Sacar hilo
		mutexExit.Lock()

		for ColaExit.EstaVacia() {
			colaExitCond.Wait() // Espera hasta que haya un hilo en ColaExit
		}

		// Se saca el hilo de Exit
		hilo := ColaExit.Dequeue()
		//slog.Debug("Se saco el hilo de la cola de exit", "TID", hilo.TID)
		mutexExit.Unlock()
		// Encontrar PCB
		pcbAsociado := punteroPCBporTCB(hilo)
		if pcbAsociado == nil {
			slog.Error("No se encontró un PCB asociado para el hilo", "TID", hilo.TID)
			mutexExit.Unlock()
			continue
		}
		// slog.Debug("Se encontro el proceso asociado al hilo", "PID", pcbAsociado.PID)
		// Funcion que finaliza hilos
		// slog.Debug("------------------------Se esta por enviar la peticion de finalizar a memoria ", "PID", pcbAsociado.PID, "TID", hilo.TID)
		FinalizarHilo(pcbAsociado, hilo.TID) // Como ya es puntero no es necesario el &

	}
}

func FinalizarHilo(proceso *PCB, tid int) {
	//slog.Debug("Antes de enviar la peticion a memoria")

	mutexFinalizacion.Lock()
	estado := PeticionFinalizarHiloAMemoria(proceso.PID, tid)

	// slog.Debug("Estado de la finalizacion del hilo: ", "Estado", estado)

	if estado == 1 { //Estado 1: Hilo finalizado correctamente en memoria, es decir se pudo liberar la memoria asociada a ese hilo

		index := -1
		for i, t := range proceso.TIDs {
			if t == tid {
				index = i
				break
			}
		}

		if index == -1 {
			slog.Error("El TID no se encontró en el array de TIDs del proceso", "TID", tid)
			mutexFinalizacion.Unlock()
			return
		}

		// Remover el TID del slice
		proceso.TIDs = append(proceso.TIDs[:index], proceso.TIDs[index+1:]...)

		mutexFinalizacion.Unlock()

		// Chekear si el hilo que se elimina esta dentro de los bloqueados por threadjoin. Si esta dentro, al finalizar agregar
		hiloAdesbloquear, existe := MapaBlockedJoin[tid]

		// Comprobar que haya hilo
		if existe && hiloAdesbloquear.PID == proceso.PID {
			mutexReady.Lock()
			ColaReady.Enqueue(hiloAdesbloquear)
			colaReadyCond.Signal() // Notifica que se añadió un hilo a la cola
			mutexReady.Unlock()
			slog.Debug("Se desbloqueo el hilo bloqueado por Join", "Hilo Desbloqueaado", hiloAdesbloquear.TID)
		}

		// slog.Debug("Largo Plazo - Se finalizo el hilo -", "TID", tid)
		slog.Info(fmt.Sprintf("## (%d:%d) Finaliza el hilo", proceso.PID, tid))

	} else {
		slog.Error("Largo Plazo - No se pudo finalizar el hilo")
		mutexFinalizacion.Unlock()
	}
}

func FinalizarProceso(proceso *PCB) {

	estado := PeticionFinalizarProcesoAMemoria(proceso.PID)

	switch estado {
	case 1:
		if len(proceso.TIDs) == 0 {
			slog.Debug(fmt.Sprintf("## El proceso %d no tiene hilos asociados. Finalizando directamente.", proceso.PID))
			mutexFinalizacion.Lock()
			ColaProcesosAux := utils.Cola[PCB]{}
			//Eliminar el proceso de la cola de proceso
			for !ColaProcesos.EstaVacia() {
				procesoCola := ColaProcesos.Dequeue()
				if !(procesoCola.PID == proceso.PID) {
					//Matar al proceso :(
					ColaProcesosAux.Enqueue(procesoCola)
				}
			}
			// slog.Debug("Se finalizo el Proceso:", "PID", proceso.PID)
			slog.Info(fmt.Sprintf("## Finaliza el proceso %d", proceso.PID))
			ColaProcesos.Elementos = ColaProcesosAux.Elementos

			mutexMemoria.Lock()
			memCond.Signal()
			mutexMemoria.Unlock()

			mutexFinalizacion.Unlock()
		} else {

			// Mover a exit todos los hilos asociados a este proceso
			for _, tid := range proceso.TIDs {
				// Proceso de la cola ColaReady
				auxColaReady := utils.Cola[TCB]{}

				mutexProcesos.Lock()

				for !ColaReady.EstaVacia() {
					hilo := ColaReady.Dequeue()                     // Saca un hilo de la cola
					if hilo.PID == proceso.PID && hilo.TID == tid { // Se fija si está asociado al proceso
						// Mueve el hilo a Exit
						mutexExit.Lock()
						ColaExit.Enqueue(hilo)
						colaExitCond.Signal() // Notifica que se añadió un hilo
						mutexExit.Unlock()
					} else {
						// Si el Hilo no se finaliza, se agrega a la cola auxiliar
						auxColaReady.Enqueue(hilo)
					}
				}
				// Reasigna la cola ColaReady con los hilos que no se eliminaron
				ColaReady.Elementos = auxColaReady.Elementos

				mutexProcesos.Unlock()

			}
			mutexFinalizacion.Lock()
			ColaProcesosAux := utils.Cola[PCB]{}
			//Eliminar el proceso de la cola de proceso
			for !ColaProcesos.EstaVacia() {
				procesoCola := ColaProcesos.Dequeue()
				if !(procesoCola.PID == proceso.PID) {
					//Matar al proceso :(
					ColaProcesosAux.Enqueue(procesoCola)
				}
			}
			// slog.Debug("Se finalizo el Proceso:", "PID", proceso.PID)
			slog.Info(fmt.Sprintf("## Finaliza el proceso %d", proceso.PID))
			ColaProcesos.Elementos = ColaProcesosAux.Elementos

			mutexMemoria.Lock()
			memCond.Signal()
			mutexMemoria.Unlock()

			mutexFinalizacion.Unlock()
		}
	case 0:
		slog.Debug("Largo Plazo - No se pudo finalizar el proceso a memoria")
	default:
		slog.Error("Largo Plazo - Valor desconocido desde memoria al finalizar proceso", "Estado", estado)
	}
}

func PeticionFinalizarProcesoAMemoria(pid int) int {
	cliente := &http.Client{}

	url := fmt.Sprintf("http://%s:%d/memoria/finalizarproceso", config.IPMemory, config.PortMemory)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return -1
	}

	q := req.URL.Query()
	q.Add("PID", strconv.Itoa(pid))
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

	est := string(bodyBytes)
	estadoPeticion, _ := strconv.Atoi(est)

	switch estadoPeticion {
	case 1:
		//slog.Debug("Terminaste todo Rey, la rompiste, idolo, mostro")
		return 1
	case 0:
		return 0
	default:
		slog.Error("Largo Plazo - Valor desconocido desde memoria al finalizar proceso", "Estado", estadoPeticion)
		return -1
	}
}

type RequestFinalizarHilo struct {
	PID int `json:"PID"`
	TID int `json:"TID"`
}

func PeticionFinalizarHiloAMemoria(pid int, tid int) int {

	body, err := json.Marshal(RequestFinalizarHilo{
		PID: pid,
		TID: tid,
	})
	if err != nil {
		return -1
	}

	cliente := &http.Client{}
	url := fmt.Sprintf("http://%s:%d/memoria/finalizarhilo", config.IPMemory, config.PortMemory)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return -1
	}

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

	est := string(bodyBytes)
	estadoCreacion, _ := strconv.Atoi(est)

	//slog.Debug("Largo Plazo - Se envio la peticion a memoria para finalizar hilo")

	return estadoCreacion
}

/* ---------------- Dump Memory ---------------- */

func PeticionDumpMemory(pid int, tid int) int {

	cliente := &http.Client{}
	url := fmt.Sprintf("http://%s:%d/memoria/dumpmemory/", config.IPMemory, config.PortMemory)
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

	est := string(bodyBytes)
	estadoPeticion, _ := strconv.Atoi(est)

	return estadoPeticion
}

/* ---------------- Funciones Auxiliares ---------------- */
func punteroPCBporTCB(tcb TCB) *PCB {
	// Esta funcion devuelve un puntero al proceso que sigue estando en la cola, no lo saca
	// Busca en ColaNew
	//slog.Debug("Entro a a la func punteroPCBporTCB")
	mutexNew.Lock()
	//slog.Debug("Post mutex de new, hilo a buscar: ", "Hilo", tcb)
	for i := range ColaNew.Elementos { // Recorre los elementos (Procesos en este caso) de la cola por índice
		if ColaNew.Elementos[i].PID == tcb.PID {
			mutexNew.Unlock()
			//slog.Debug("Encontro el proceso en la cola de new", "Proceso", ColaProcesos.Elementos[i])
			return &ColaNew.Elementos[i] // Devuelve el puntero al Proceso encontrado
		}
	}

	mutexNew.Unlock()
	//slog.Debug("Post mutex de new, no encontro el proceso en la cola de new")

	mutexReady.Lock()
	//slog.Debug("Post mutex de ready, hilo a buscar: ", "Hilo", tcb)
	// Busca en ColaProcesos
	for i := range ColaProcesos.Elementos {
		if ColaProcesos.Elementos[i].PID == tcb.PID {
			mutexReady.Unlock()
			//slog.Debug("Encontro el proceso en la cola de procesos", "Proceso", ColaProcesos.Elementos[i])
			return &ColaProcesos.Elementos[i]
		}
	}

	mutexReady.Unlock()

	//slog.Debug("No encontro el proceso en ninguna cola")

	return nil // Si no encontró el PCB asociado

}

/* ------------------------------------------------------ */
