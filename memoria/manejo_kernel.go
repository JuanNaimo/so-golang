package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strconv"
)

/*** --------------- Manejo de Largo Plazo de Kernel (Crear y Finalizar Procesos e Hilos) --------------- ***/

/* ------- Funciones de Crear -------*/
func CrearProcesoMemoria(w http.ResponseWriter, r *http.Request) {
	var request RequestPeticion
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pid := request.PID
	tamanioProceso := request.TamanioProceso

	slog.Debug("Se recibio una peticion para crear proceso", "PID", pid, "Tamaño", tamanioProceso)

	// Buscar Particion Por algoritmo
	var punteroParticion *Particion
	seEncontro := false

	if config.SearchAlgorithm == "FIRST" {
		punteroParticion, seEncontro = buscarPorFirst(tamanioProceso)

	} else if config.SearchAlgorithm == "BEST" {
		punteroParticion, seEncontro = buscarPorBest(tamanioProceso)

	} else if config.SearchAlgorithm == "WORST" {
		punteroParticion, seEncontro = buscarPorWorst(tamanioProceso)
	}

	// Asignar registros (BASE Y LIMITE TAMBIEN)
	// fijo
	if config.Scheme == "FIJAS" {

		if !seEncontro { // Si no hay espacio, devolver un 0
			slog.Debug("No se encontro particion para el proceso", "PID", pid)
			respuesta, _ := json.Marshal(0)
			w.WriteHeader(http.StatusOK)
			w.Write(respuesta)
			return
		}

		punteroParticion.Ocupado = true
		MapaPidContexto[pid] = BaseYLimite{PunteroParticion: punteroParticion, Base: punteroParticion.Base, Limite: punteroParticion.Base + punteroParticion.Tamanio - 1}
		slog.Debug("Pid asignado a particion, ", "PID", pid, "Particion", punteroParticion, "Mapa Actualizado", MapaPidContexto[pid])

	}

	// dinamico
	if config.Scheme == "DINAMICAS" {
		if !seEncontro { // Si no hay espacio, devolver un 0
			// Calcular si la sumatoria de los espacios libres alcanza para el tamanio del proceso
			totalLibre := calcularParticionesLibres()
			// Si no alcanza, devolver un 0
			if totalLibre < uint32(tamanioProceso) {
				respuesta, _ := json.Marshal(0)
				w.WriteHeader(http.StatusOK)
				w.Write(respuesta)
				return
			}
			// Si alcanza, compactar y asignar ese espacio al proceso
			compactarMemoria()

			// A BORRAR: Es para mostrar como estan los mapas despues de conseguir una particion
			// for indice, particion := range MapaParticiones {
			// 	slog.Debug("POST COMPACTAR - Particion del mapa de particiones", "Numero", indice, "Base", particion.Base, "Limite", particion.Base+particion.Tamanio-1, "Ocupado", particion.Ocupado)
			// }

			// Buscar nuevamente una partición para el proceso
			if config.SearchAlgorithm == "FIRST" {
				punteroParticion, _ = buscarPorFirst(tamanioProceso)

			} else if config.SearchAlgorithm == "BEST" {
				punteroParticion, _ = buscarPorBest(tamanioProceso)

			} else if config.SearchAlgorithm == "WORST" {
				punteroParticion, _ = buscarPorWorst(tamanioProceso)
			}
		}

		punteroParticion.Ocupado = true                      // Actual
		tamanioParticionOriginal := punteroParticion.Tamanio // Restante

		punteroParticion.Tamanio = uint32(tamanioProceso) // Actual

		restante := tamanioParticionOriginal - uint32(tamanioProceso) // Restante

		nuevaBase := punteroParticion.Base + uint32(tamanioProceso) // Restante

		MapaPidContexto[pid] = BaseYLimite{PunteroParticion: punteroParticion, Base: punteroParticion.Base, Limite: punteroParticion.Base + punteroParticion.Tamanio - 1} // Actual

		if restante != 0 {
			MapaParticiones[int(nuevaBase)] = &Particion{Ocupado: false, Base: nuevaBase, Tamanio: restante} // Restante
		}

		slog.Debug("Memoria Kernel - tamanio restante de memoria -", "Tamanio", restante)
	}

	// A BORRAR: Es para mostrar como estan los mapas despues de conseguir una particion
	// for indice, particion := range MapaParticiones {
	// 	slog.Debug("Particion del mapa de particiones", "Numero", indice, "Base", particion.Base, "Limite", particion.Base+particion.Tamanio-1, "Ocupado", particion.Ocupado)
	// }

	slog.Info(fmt.Sprintf("## Proceso Creado -  PID: %d - Tamaño: %d", pid, tamanioProceso))
	respuesta, err := json.Marshal(1)
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

func CrearHiloMemoria(w http.ResponseWriter, r *http.Request) {
	var request RequestPeticionHilo
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pid := request.PID
	tid := request.TID
	ruta := request.RutaPseudocodigo
	//slog.Debug("--------------- Ruta Recibida desde kernel", "Ruta", ruta)
	//slog.Debug("Ingreso a crear hilo", "Hilo", tid)
	errr := cargarPseudoCodigoTid(ruta, pid, tid)
	var registros RegistroCPU
	registros.AX = 0
	registros.BX = 0
	registros.CX = 0
	registros.DX = 0
	registros.EX = 0
	registros.FX = 0
	registros.GX = 0
	registros.HX = 0
	registros.PC = 0

	MapaTidContexto[PidYTid{PID: pid, TID: tid}] = registros

	var respuesta []byte
	if errr != nil {
		slog.Error("Error al crear hilo - ", "Error", errr)
		respuesta, _ = json.Marshal(0)
	} else {
		respuesta, _ = json.Marshal(1)
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
	slog.Info(fmt.Sprintf("## Hilo Creado - (PID:TID) - (%d:%d)", pid, tid))
	//slog.Debug("Memoria Kernel - Se creo Hilo")
}

func cargarPseudoCodigoTid(ruta string, pid int, tid int) error {
	// Abrimos el archivo
	//slog.Debug("Entro a cargar pseudocodigo", "Ruta", ruta, "PID", pid, "TID", tid)
	//slog.Debug(ruta)
	archivo, err := os.Open(ruta)
	if err != nil {
		return err
	}
	defer archivo.Close()
	//Para que no exista la instruccion 0 en el mapa
	MapaInstrucciones[PidYTid{TID: tid, PID: pid}] = append(MapaInstrucciones[PidYTid{TID: tid, PID: pid}], "Instruccion 0")

	scanner := bufio.NewScanner(archivo)
	for scanner.Scan() {
		linea := scanner.Text()
		MapaInstrucciones[PidYTid{TID: tid, PID: pid}] = append(MapaInstrucciones[PidYTid{TID: tid, PID: pid}], linea)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	//slog.Debug("Memoria Kernel - Se cargo seudocodigo")

	return nil
}

/* ----- Funciones de Finalizar -----*/
func FinalizarProcesoMemoria(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	pidRecibido := queryParams.Get("PID")
	pid, _ := strconv.Atoi(pidRecibido)

	slog.Debug("Contexto del Mapa", "PID", pid, "Contexto", MapaPidContexto[pid], "Mapa Entero", MapaPidContexto)

	if contextoPid, encontrado := MapaPidContexto[pid]; encontrado {
		// Marca la partición asociada como libre si existe
		slog.Debug("Entro al if raro de encontrado")
		if contextoPid.PunteroParticion != nil {
			contextoPid.PunteroParticion.Ocupado = false
		}
	}

	// Habria que agarrar la base de la particion que se libero y restarle 1 para ver si hay una particion libre a la izquierda y lo mismo sumarle 1 al limite
	// para ver si hay una particion libre a la derecha.
	// Si hay una particion libre, crear una nueva particion con la suma de los dos espacios libres.
	particionLiberadaMapa := MapaPidContexto[pid]
	//slog.Debug("Particion Liberada Mapa", "")
	particionLiberada := particionLiberadaMapa.PunteroParticion

	tamanioAliberar := particionLiberada.Tamanio
	baseParticionLiberada := particionLiberada.Base
	limiteParticionLiberada := particionLiberada.Base + particionLiberada.Tamanio - 1

	if config.Scheme == "DINAMICAS" {

		// Variables para rastrear nuevas bases y límites
		nuevaBase := baseParticionLiberada
		nuevoLimite := limiteParticionLiberada

		// Recorrer las particiones y verificar adyacentes
		for id, particion := range MapaParticiones {
			// Si es una partición libre
			if !particion.Ocupado {
				// Verificar si está a la izquierda
				if particion.Base+particion.Tamanio == baseParticionLiberada {
					nuevaBase = particion.Base
					delete(MapaParticiones, int(baseParticionLiberada))
					delete(MapaParticiones, id) // Eliminar la partición combinada
				}

				// Verificar si está a la derecha
				if baseParticionLiberada+particionLiberada.Tamanio == particion.Base {
					nuevoLimite = particion.Base + particion.Tamanio - 1
					delete(MapaParticiones, int(baseParticionLiberada))
					delete(MapaParticiones, id) // Eliminar la partición combinada
				}
			}
		}

		// Crear una nueva partición combinada
		slog.Debug("Borramos una particion, vamos a crear la siguiente con", "Base", nuevaBase, "Limite", nuevoLimite)
		nuevoTamanio := nuevoLimite - nuevaBase + 1
		MapaParticiones[int(nuevaBase)] = &Particion{
			Ocupado: false,
			Base:    nuevaBase,
			Tamanio: nuevoTamanio,
		}

		// for indice, particion := range MapaParticiones {
		// 	slog.Debug("Particion del mapa de particiones", "Numero", indice, "Base", particion.Base, "Limite", particion.Base+particion.Tamanio-1, "Ocupado", particion.Ocupado)
		// }
	}

	slog.Debug("Memoria Kernel - Se finalizo el proceso", "PID", pid)
	slog.Info(fmt.Sprintf("## Proceso Destruido -  PID: %d - Tamaño: %d", pid, tamanioAliberar))
	respuesta, _ := json.Marshal(1)

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

func FinalizarHiloMemoria(w http.ResponseWriter, r *http.Request) {
	var request RequestFinalizarHilo
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	pid := request.PID
	tid := request.TID

	// Aca se deberia borrar el hilo del mapatidcontexto, que tiene los registros del cpu
	delete(MapaTidContexto, PidYTid{PID: pid, TID: tid})
	//slog.Debug("--------Se borro el hilo del mapaTidContexto", "Tid", tid, "Nuevo estado", MapaTidContexto)
	// Borra el hilo del mapa de instrucciones
	delete(MapaInstrucciones, PidYTid{PID: pid, TID: tid})

	//slog.Debug("--------Se borro el hilo del mapaInstrucciones", "Tid", tid, "Nuevo estado", MapaInstrucciones)

	respuesta, err := json.Marshal(1)
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}
	////slog.Debug("Memoria Kernel - Se finalizo Hilo", "Hilo", tid)
	slog.Info(fmt.Sprintf("## Hilo Destruido - (PID:TID) - (%d:%d)", pid, tid))
	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

/* ------ Funciones de Dump Memory -------*/
func AtenderDumpMemory(w http.ResponseWriter, r *http.Request) {

	queryParams := r.URL.Query()
	pidRecibido := queryParams.Get("PID")
	tidRecibido := queryParams.Get("TID")
	pid, _ := strconv.Atoi(pidRecibido)
	tid, _ := strconv.Atoi(tidRecibido)

	//Aca se deberia de leer el array de memoria y enviarselo al fs, todavia no tiene nada dentro porque no lo cambiamos
	nombre := fmt.Sprintf("%s-%s", pidRecibido, tidRecibido)

	//Busco el proceso en el mapa para ver la base y el limite y leer esos bytes en el espacio de memoria
	proceso := MapaPidContexto[pid]
	tamanioEnarray := proceso.Limite - proceso.Base
	// Leer el contenido de la memoria
	contenido := EspacioMemoria[proceso.Base : proceso.Limite+1]

	body, err := json.Marshal(BodyDumpFile{
		FileName: nombre,
		Size:     int(tamanioEnarray),
		Content:  contenido,
	})

	if err != nil {
		return
	}

	cliente := &http.Client{}
	url := fmt.Sprintf("http://%s:%d/filesystem/dump_memory", config.IPFilesystem, config.PortFilesystem)
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

	respuestaFS, err := io.ReadAll(respuesta.Body) //Respuesta desde fs de como salio el dumpmemory
	if err != nil {
		http.Error(w, "Error al leer la respuesta del servidor remoto", http.StatusInternalServerError)
		return
	}

	slog.Info(fmt.Sprintf("## Memory Dump solicitado - (PID:TID) - (%d:%d)", pid, tid))
	slog.Debug("Respuesta del Fs", "Respuesta", string(respuestaFS))
	w.WriteHeader(http.StatusOK)
	w.Write(respuestaFS)
}

/* ------ Funciones Particiones -------*/
func buscarPorFirst(tamanio int) (*Particion, bool) {
	indices := make([]int, 0, len(MapaParticiones))
	for i, particion := range MapaParticiones {
		if !particion.Ocupado {
			indices = append(indices, i)
		}

	}

	// Ordenar las claves
	sort.Ints(indices)

	for _, i := range indices {
		particion := MapaParticiones[i]
		if !particion.Ocupado && particion.Tamanio >= uint32(tamanio) {
			return particion, true
		}
	}

	return nil, false
}

func buscarPorBest(tamanio int) (*Particion, bool) {
	var mejorParticion *Particion
	encontrado := false

	for _, particion := range MapaParticiones {
		if !particion.Ocupado && particion.Tamanio >= uint32(tamanio) {
			// Si es la primera partición encontrada o es más grande que la anterior peor
			if !encontrado || particion.Tamanio < mejorParticion.Tamanio {
				mejorParticion = particion
				encontrado = true
			}
		}
	}

	return mejorParticion, encontrado
}

func buscarPorWorst(tamanio int) (*Particion, bool) {
	var peorParticion *Particion
	encontrado := false

	for _, particion := range MapaParticiones {
		if !particion.Ocupado && particion.Tamanio >= uint32(tamanio) {
			// Si es la primera partición encontrada o es más grande que la anterior peor
			if !encontrado || particion.Tamanio > peorParticion.Tamanio {
				peorParticion = particion
				encontrado = true
			}
		}
	}

	return peorParticion, encontrado
}

/* ------ Funciones Generales -------*/

func calcularParticionesLibres() uint32 {
	var suma uint32 = 0

	for _, particion := range MapaParticiones {
		if !particion.Ocupado {
			suma += particion.Tamanio
		}
	}

	return suma
}

/* ------------- Funciones de Compactar Memoria ------------- */

func compactarMemoria() {
	var nuevaBase uint32 = 0
	particionesCompactadas := make(map[int]*Particion)

	// Iterar sobre las particiones para mover las ocupadas al inicio de la memoria
	for _, particion := range MapaParticiones {
		if particion.Ocupado {
			// Mover la partición ocupada al inicio de la memoria
			copy(EspacioMemoria[nuevaBase:nuevaBase+particion.Tamanio], EspacioMemoria[particion.Base:particion.Base+particion.Tamanio])
			particion.Base = nuevaBase
			particionesCompactadas[int(nuevaBase)] = particion
			nuevaBase += particion.Tamanio // Actualizar la base para la siguiente partición ocupada
		}
	}

	// Crear una partición libre que ocupe el resto del espacio
	tamanoLibre := uint32(len(EspacioMemoria)) - nuevaBase
	if tamanoLibre > 0 {
		nuevaParticionLibre := &Particion{
			Ocupado: false,
			Base:    nuevaBase,
			Tamanio: tamanoLibre,
		}
		particionesCompactadas[int(nuevaBase)] = nuevaParticionLibre
	}

	// Reemplazar MapaParticiones con el mapa compactado
	MapaParticiones = particionesCompactadas

	slog.Debug("Memoria compactada. Todas las particiones libres se combinaron en una sola al final de la memoria.")
}
