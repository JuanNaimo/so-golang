package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/sisoputnfrba/tp-golang/utils"
)

/* ----------- Variables globales de kernel centradas en corto plazo ----------- */
// Variables
var tidEjecutando int
var pidEjecutando int
var hiloAProcesar TCB

// Colas
var colasPrioridad = make(map[int]*utils.Cola[TCB])

// Semaforos
var colaReadyCond = sync.NewCond(&mutexReady)
var colaProcesosCond = sync.NewCond(&mutexProcesos)
var condPrioridad = sync.NewCond(&mutexPrioridad)

/* ----------- Funciones de Corto Plazo ----------- */

func PlanificadorCortoPlazo() {

	for {
		// Semaforos de Procesos y Ready (Condicional)
		mutexProcesos.Lock()

		for ColaProcesos.EstaVacia() {
			colaProcesosCond.Wait() // Espera hasta que haya procesos en las colas
		}

		mutexProcesos.Unlock()

		mutexReady.Lock()
		for ColaReady.EstaVacia() && config.SchedulerAlgorithm == "FIFO" {
			colaReadyCond.Wait() // Espera hasta que haya procesos en las colas
		}

		mutexReady.Unlock()

		slog.Debug("Corto Plazo - Hay procesos en Ready")

		algoritmoPlanificador := config.SchedulerAlgorithm // Traer la variable del algoritmo del config

		hiloAProcesar = TCB{} // Vaciamos el hilo a procesar

		// Si es fifo -> Sacar el hilo por fifo
		if algoritmoPlanificador == "FIFO" {
			mutexReady.Lock()
			hiloAProcesar = ColaReady.Dequeue()
			mutexReady.Unlock()
			slog.Debug("Corto Plazo - FIFO Dequeue", "TID", hiloAProcesar.TID)
		}

		// Si es Prioridad o CMN -> Sacar el hilo de Ready y asignarlo a cola de prioridad
		if algoritmoPlanificador == "PRIORIDAD" || algoritmoPlanificador == "CMN" {
			//slog.Debug("Estado cola de prioridad - Previa", "Cola", colasPrioridad)
			//slog.Debug("Cola Ready - Previa", "Cola", ColaReady)
			// Colas de Prioridades
			mutexReady.Lock()
			for !ColaReady.EstaVacia() {
				hiloPrioridad := ColaReady.Dequeue()
				nivelPrioridad := hiloPrioridad.Prioridad
				ingresarColaPrioridad(nivelPrioridad, hiloPrioridad)
				mutexPrioridad.Lock()
				condPrioridad.Signal()
				mutexPrioridad.Unlock()
			}
			mutexReady.Unlock()
			//slog.Debug("Cola Ready - Post", "Cola", ColaReady)
			//slog.Debug("Estado cola de prioridad - post", "Cola", colasPrioridad)
		}

		// Si es Prioridad o CMN -> Obtener el hilo de mayor prioridad
		if algoritmoPlanificador == "PRIORIDAD" || algoritmoPlanificador == "CMN" {
			hiloAProcesar = hiloMayorPrioridad() // Selecciona el hilo de mayor prioridad

			if hiloAProcesar == (TCB{}) {
				slog.Debug("No hay hilos disponibles para ejecutar. Esperando...")
				continue
			}

			slog.Debug("Corto Plazo - PRIORIDAD Dequeue", "TID", hiloAProcesar.TID, "PID", hiloAProcesar.PID)
		}

		// Cambiar variables globales con el hilo a ejecutar
		tidEjecutando = hiloAProcesar.TID
		pidEjecutando = hiloAProcesar.PID
		tcbEjecutando = hiloAProcesar
		pcbEjecutando = *punteroPCBporTCB(tcbEjecutando)

		if tidEjecutando < 0 {
			//slog.Error("Corto Plazo - Error al seleccionar hilo", "TID", tidEjecutando, "PID", pidEjecutando)
			return
		}
		// slog.Debug("Corto Plazo - Hilo Seleccionado a ejecutar", "TID", tidEjecutando, "PID", pidEjecutando)

		// Si es CMN -> Preparar Quantumm
		if config.SchedulerAlgorithm == "CMN" {
			go func() {
				time.Sleep(time.Duration(config.Quantum) * time.Millisecond) // RR ♥♥
				slog.Debug("Corto Plazo - CMN Dequeue", "TID", hiloAProcesar.TID, "PID", hiloAProcesar.PID)
				// Enviar quantum con tid
				EjecutarInterrupcion("QUANTUM", hiloAProcesar.TID)
				// slog.Debug("Corto Plazo - Interrupción enviada al CPU por quantum", "Hilo", hiloAProcesar.TID)
			}()
		}

		slog.Debug(" ---- > Se envia un hilo a cpu", "TID", hiloAProcesar.TID, "PID", hiloAProcesar.PID)

		// Enviar al cpu el tid y pid
		enviarACpu(tidEjecutando, pidEjecutando)
	}
}

func enviarACpu(tid int, pid int) {
	cliente := &http.Client{}
	url := fmt.Sprintf("http://%s:%d/cpu/proceso", config.IPCPU, config.PortCPU)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return
	}

	q := req.URL.Query()
	q.Add("PID", strconv.Itoa(pid))
	q.Add("TID", strconv.Itoa(tid))
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Content-Type", "application/json")
	respuesta, err := cliente.Do(req)
	if err != nil {
		return
	}

	// Verificar el código de estado de la respuesta
	if respuesta.StatusCode != http.StatusOK {
		return
	}

	bodyBytes, err := io.ReadAll(respuesta.Body)
	if err != nil {
		return
	}

	slog.Debug(string(bodyBytes))
}

func manejarMotivoDesalojo(motivo string) {
	slog.Debug("Corto Plazo - Se recibio respuesta del CPU", "Motivo de Desalojo", motivo)

	switch motivo {
	case "PIOLA":
		slog.Debug("Se finalizo piola el hilo")
	case "PRIORIDAD":
		mutexReady.Lock()
		// slog.Debug("Corto Plazo - Desalojo Proridad", "PID", hiloAProcesar.PID, "Prioridad", hiloAProcesar.Prioridad)
		// slog.Debug("Corto Plazo - Info del ejecutar", "PID", tcbEjecutando.PID, "Prioridad", tcbEjecutando.Prioridad)
		hiloAProcesar.Estado = Ready
		ingresarColaPrioridad(hiloAProcesar.Prioridad, hiloAProcesar)
		// slog.Debug("Estado cola de prioridad", "Cola", colasPrioridad)
		mutexReady.Unlock()
	case "QUANTUM":
		// slog.Debug("Corto Plazo - Se entro al case quantum")
		slog.Info(fmt.Sprintf("## (%d:%d) - Desalojado por fin de Quantum", hiloAProcesar.PID, hiloAProcesar.TID))
		mutexReady.Lock()
		hiloAProcesar.Estado = Ready
		ingresarColaPrioridad(hiloAProcesar.Prioridad, hiloAProcesar)
		mutexReady.Unlock()
	case "SYSCALL":
		//slog.Debug("Corto Plazo - Atendiendo SYSCALL")
	case "Segmentation Fault":
		//Finalizar proceso
		//slog.Debug("Corto Plazo - Atendiendo Segmentation Fault")
		sysPROCCESS_EXIT()
	default:
		slog.Error("Corto Plazo - Error al manejar el motivo de desalojo (Motivo desconocido)", "Motivo", motivo)
	}
}

func SeleccionarHiloPrioridad() {
	for {
		// Semaforo Condicional
		mutexPrioridad.Lock()

		for hiloAProcesar.Estado != Running {
			condPrioridad.Wait()
		}

		//Hilo de mayor prioridad dentro de los que estan en sus respectivas colas
		hiloMayorPrioridad := DefinirPrioridad(hiloAProcesar)
		if hiloMayorPrioridad != hiloAProcesar {
			EjecutarInterrupcion("PRIORIDAD", tidEjecutando)
		}
		mutexPrioridad.Unlock()
	}
}

func DefinirPrioridad(hiloActual TCB) TCB {
	// Recorrer el mapa buscando hilos en colas de mayor prioridad
	for prioridad := 0; prioridad < hiloActual.Prioridad; prioridad++ {
		cola := colasPrioridad[prioridad]
		if !cola.EstaVacia() {
			// Desalojar el hilo actual si hay uno en la cola de mayor prioridad
			hiloActual.Estado = Ready
			ColaReady.Enqueue(hiloActual)
			colaReadyCond.Signal() // Avisa que hay un hilo en la cola

			// Obtener el hilo de la cola de mayor prioridad
			nuevoHilo := cola.Dequeue()
			nuevoHilo.Estado = Running

			slog.Debug(fmt.Sprintf("Desalojando hilo TID %d de prioridad %d. Ejecutando hilo TID %d de prioridad %d\n", hiloActual.TID, hiloActual.Prioridad, nuevoHilo.TID, nuevoHilo.Prioridad))

			// Actualizar la cola en el mapa
			colasPrioridad[prioridad] = cola

			return nuevoHilo
		}
	}

	// Si no hay hilos de mayor prioridad, continuar con el hilo actual
	return hiloActual
}

func hiloMayorPrioridad() TCB {
	var colaResultado *utils.Cola[TCB]

	// Inicializamos la variable para comparar con la mayor prioridad.
	// Usamos el valor máximo de int para asegurarnos de encontrar la prioridad más baja.
	menorPrioridad := int(^uint(0) >> 1)

	for prioridad, cola := range colasPrioridad {
		// Verificar que la cola no esté vacía y que tenga la menor prioridad.
		if !cola.EstaVacia() && prioridad < menorPrioridad {
			slog.Debug("Estado de las colas:", "Prioridad", prioridad, "Cola", cola.Elementos)
			menorPrioridad = prioridad
			colaResultado = cola
		}
	}

	if colaResultado == nil {
		slog.Debug("Todas las colas están vacías, no hay hilos para ejecutar")
		return TCB{} // Devuelve un TCB vacío para manejarlo en el planificador
	}

	// Retornamos el puntero a la cola con la mayor prioridad no vacía, o nil si no se encontró ninguna.
	//slog.Debug("------------- Esta y me voy", "Colas", &colasPrioridad, "Cola Juguetona", colaResultado, "Prioridad", menorPrioridad)

	hiloAdevolver := colaResultado.Dequeue()
	slog.Debug("Hilo a devolver", "Hilo", hiloAdevolver)
	return hiloAdevolver
}

// Chequear si existe la cola de prioridad
// Si no existe la cola de prioridad, se crea. Si existe, se agrega
func ingresarColaPrioridad(nivel int, hilo TCB) {
	cola, existe := colasPrioridad[nivel]
	if existe {
		slog.Debug("Corto Plazo - Agregando hilo a cola de prioridad", "Hilo", hilo.TID, "Prioridad", nivel)
		cola.Enqueue(hilo)
		slog.Debug("Corto Plazo - Estado despues de agregar el hilo: ", "Cola Nueva", cola, "Cantidad Elementos", len(cola.Elementos))
	} else {
		// Si no existe la cola, crear una nueva
		nuevaCola := &utils.Cola[TCB]{}
		nuevaCola.Enqueue(hilo)
		slog.Debug("Corto Plazo - Nueva cola de prioridad creada", "Prioridad", nivel)
		// Asignar la nueva cola al mapa
		colasPrioridad[nivel] = nuevaCola
	}
}

func EjecutarInterrupcion(motivo string, tid int) {

	cliente := &http.Client{}
	url := fmt.Sprintf("http://%s:%d/cpu/interrupt", config.IPCPU, config.PortCPU)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		slog.Error("Error")
	}
	q := req.URL.Query()
	q.Add("TID", strconv.Itoa(tid))
	q.Add("MOTIVO", motivo)
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
	slog.Debug("--------------------------------------------Corto Plazo - Interrupción enviada al CPU", "Motivo", motivo)
}

func AtenderContextoDesalojo(w http.ResponseWriter, r *http.Request) {

	var request BodyDesalojo

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	motivo := request.Motivo

	manejarMotivoDesalojo(motivo)
}
