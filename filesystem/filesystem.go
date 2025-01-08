package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/sisoputnfrba/tp-golang/utils"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Iniciar Configuracion
	err := utils.IniciarConfiguracion(&config)
	if err != nil {
		fmt.Println("Error al cargar la configuración del FileSystem:", err)
		return
	}

	// Definir nivel de los logs
	slog.SetLogLoggerLevel(utils.LogLevels[config.LogLevel])
	utils.IniciarLogger()

	// Iniciar Servidor
	utils.RegistrarRutas(rutas)

	go func() {
		slog.Info("Se inicio el servidor de fileSystem", "puerto", config.Port)
		panic(http.ListenAndServe(":"+strconv.Itoa(config.Port), nil))
	}()

	// Inicializar el sistema de archivos (usando la v2)
	filesystem, err = inicializarFileSystem()
	if err != nil {
		log.Printf("Error inicializando el sistema de archivos: %v", err)
		return
	}

	defer func() {
		guardarBitmapErr := guardarBitmap(filesystem.Bitmap, filepath.Join(config.MountDir, "bitmap.dat"))
		slog.Debug("Se guardaron los bitmap")
		if guardarBitmapErr != nil {
			log.Printf("Error al guardar el bitmap: %v", guardarBitmapErr)
		}
		filesystem.Bloques.Close()
	}()

	<-ctx.Done()
	//select {}
}

/* --------- Manejo de Sistema de Archivos --------- */

// V2
func inicializarFileSystem() (*FileSystem, error) {
	fs := &FileSystem{
		Bitmap:   make([]byte, (config.BlockCount+7)/8), // Un bit por bloque
		Metadata: make(map[string]*FileMetadata),
	}

	// Rutas de los archivos principales
	rutaBitmap, _ := filepath.Abs(filepath.Join(config.MountDir, "/bitmap.dat"))
	rutaBloques, _ := filepath.Abs(filepath.Join(config.MountDir, "/bloques.dat"))

	// Crear o abrir bitmap.dat
	bitmapFile, err := crearOAbrirArchivo(rutaBitmap, len(fs.Bitmap))
	if err != nil {
		return nil, fmt.Errorf("error con bitmap.dat: %v", err)
	}
	defer bitmapFile.Close()

	// Leer el estado del bitmap
	if _, err := bitmapFile.Read(fs.Bitmap); err != nil && err != io.EOF {
		return nil, fmt.Errorf("error al leer bitmap.dat: %v", err)
	}

	// Crear o abrir bloques.dat
	bloquesFile, err := crearOAbrirArchivo(rutaBloques, config.BlockCount*config.BlockSize)
	if err != nil {
		return nil, fmt.Errorf("error con bloques.dat: %v", err)
	}
	fs.Bloques = bloquesFile

	return fs, nil
}

func guardarBitmap(bitmap []byte, rutaBitmap string) error {
	ruta, _ := filepath.Abs(rutaBitmap)
	archivo, err := os.Create(ruta)
	if err != nil {
		return fmt.Errorf("error al crear archivo bitmap.dat: %v", err)
	}
	defer archivo.Close()

	_, err = archivo.Write(bitmap)
	if err != nil {
		return fmt.Errorf("error al escribir bitmap.dat: %v", err)
	}
	return nil
}

// Función para limpiar mount_dir/files de ser necesario
// func limpiarArchivosDump() error {
// 	carpetaFiles, _ := filepath.Abs(filepath.Join(config.MountDir, "files"))
// 	entradas, err := os.ReadDir(carpetaFiles)
// 	slog.Debug("Inicio de eliminar los archivos", "Directorio", carpetaFiles)
// 	if err != nil {
// 		slog.Error("Error: no se pudo leer las entradas de la carpeta")
// 		return err
// 	}

// 	for _, entrada := range entradas {
// 		ruta := filepath.Join(carpetaFiles, entrada.Name())
// 		slog.Debug("Se va a eliminar", "Directorio", ruta)

// 		err := os.Remove(ruta)
// 		if err != nil {
// 			return err
// 		}

// 	}

// 	return nil
// }

/* ---------- Manejo de Archivos ---------- */

// Función para crear o abrir un archivo si ya existe
func crearOAbrirArchivo(ruta string, tamano int) (*os.File, error) {

	slog.Debug("Creando o abriendo archivo", "ruta", ruta, "tamano", tamano)

	archivo, err := os.OpenFile(ruta, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	// Verificar el tamaño del archivo
	info, err := archivo.Stat()
	if err != nil {
		archivo.Close()
		return nil, err
	}

	if info.Size() != int64(tamano) {
		// Si el tamaño del archivo no es el esperado, redimensionarlo
		err = archivo.Truncate(int64(tamano))
		if err != nil {
			archivo.Close()
			return nil, err
		}
	}

	return archivo, nil
}

/* ---------- Manejo de Dump ---------- */
// Creación de archivo de DUMP
func CrearArchivoDump(w http.ResponseWriter, r *http.Request) {
	// Decodificar la solicitud de Memoria
	var request BodyDumpFile
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		slog.Debug("Error al decodificar la solicitud", "error", err)
		respuesta, _ := json.Marshal(0)

		w.WriteHeader(http.StatusOK)
		w.Write(respuesta)
		return
	}

	// Verificar si hay espacio suficiente
	if !verificarEspacioDisponible(request.Size) {
		slog.Debug("No hay espacio suficiente para el archivo", "size", request.Size)
		slog.Info(fmt.Sprintf("## Fin de solicitud - Archivo: %s", request.FileName))
		respuesta, _ := json.Marshal(0)

		w.WriteHeader(http.StatusOK)
		w.Write(respuesta)
		return
	}

	// Reservar bloques
	bloques, err := reservarBloques(request.Size, request.FileName)
	if err != nil {
		slog.Debug("Error al reservar bloques", "error", err)
		slog.Info(fmt.Sprintf("## Fin de solicitud - Archivo: %s", request.FileName))
		respuesta, _ := json.Marshal(0)

		w.WriteHeader(http.StatusOK)
		w.Write(respuesta)
		return
	}

	// Crear metadata
	if err := crearMetadata(request.FileName, request.Size, bloques[0]); err != nil {
		slog.Debug("Error al crear metadata", "error", err)
		slog.Info(fmt.Sprintf("## Fin de solicitud - Archivo: %s", request.FileName))
		respuesta, _ := json.Marshal(0)

		w.WriteHeader(http.StatusOK)
		w.Write(respuesta)
		return
	}

	// Escribir punteros en el bloque índice
	if err := grabarPunteros(bloques[0], bloques[1:], request.FileName); err != nil {
		slog.Debug("Error al escribir punteros", "error", err)
		slog.Info(fmt.Sprintf("## Fin de solicitud - Archivo: %s", request.FileName))
		respuesta, _ := json.Marshal(0)

		w.WriteHeader(http.StatusOK)
		w.Write(respuesta)
		return
	}

	// Escribir contenido en los bloques de datos
	if err := escribirBloques(request.Content, bloques[1:], request.FileName); err != nil {
		slog.Debug("Error al escribir contenido", "error", err)
		slog.Info(fmt.Sprintf("## Fin de solicitud - Archivo: %s", request.FileName))
		respuesta, _ := json.Marshal(0)

		w.WriteHeader(http.StatusOK)
		w.Write(respuesta)
		return
	}

	slog.Info(fmt.Sprintf("## Archivo Creado: %s - Tamaño: %d", request.FileName, request.Size))
	slog.Info(fmt.Sprintf("## Fin de solicitud - Archivo: %s", request.FileName))
	respuesta, _ := json.Marshal(1)
	// Responder a Memoria
	//Estado 1: Creacion de archivo exitosa
	//Estado 0: Creacion de archivo fallida
	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

func verificarEspacioDisponible(size int) bool {
	bloquesNecesarios := (size + config.BlockSize - 1) / config.BlockSize
	bloquesTotales := bloquesNecesarios + 1 // +1 para el bloque índice
	espacioLibre := 0

	for _, byte := range filesystem.Bitmap {
		for bit := 0; bit < 8; bit++ {
			if byte&(1<<bit) == 0 {
				espacioLibre++
				if espacioLibre >= bloquesTotales {
					return true
				}
			}
		}
	}

	return false
}

/* ---------- Manejo de Bloques ---------- */
func reservarBloques(size int, nombre string) ([]int, error) {
	bloquesNecesarios := (size + config.BlockSize - 1) / config.BlockSize
	bloquesTotales := bloquesNecesarios + 1 // +1 para el bloque índice
	var bloquesReservados []int

	for i, byte := range filesystem.Bitmap {
		for bit := 0; bit < 8; bit++ {
			if byte&(1<<bit) == 0 { // Verificar si el bloque está libre
				filesystem.Bitmap[i] |= (1 << bit) // Marcar como ocupado
				bloquesReservados = append(bloquesReservados, i*8+bit)
				bloqAsignado := i*8 + bit
				bloquesLibres := calcularBloquesLibres(filesystem.Bitmap)
				slog.Info(fmt.Sprintf("## Bloque asignado: %d - Archivo: %s - Bloques Libres: %d", bloqAsignado, nombre, bloquesLibres))
				if len(bloquesReservados) == bloquesTotales {

					return bloquesReservados, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("no hay suficientes bloques disponibles")
}

func crearMetadata(fileName string, size int, indexBlock int) error {
	metadata := map[string]interface{}{
		"index_block": indexBlock,
		"size":        size,
	}

	tiempoFormateado := fmt.Sprintf("%02d:%02d:%02d:%03d", time.Now().Hour(), time.Now().Minute(), time.Now().Second(), time.Now().Nanosecond()/1e6)

	metadataPath, _ := filepath.Abs(filepath.Join(config.MountDir, "files", fmt.Sprintf("%s-%s.dmp", fileName, tiempoFormateado)))
	metadataFile, err := os.Create(metadataPath)
	if err != nil {
		return fmt.Errorf("error al crear metadata: %v", err)
	}
	defer metadataFile.Close()

	return json.NewEncoder(metadataFile).Encode(metadata)
}

func grabarPunteros(indexBlock int, dataBlocks []int, nombreArchivo string) error {
	// Crear un slice de bytes para almacenar los punteros
	indice := make([]byte, len(dataBlocks)*4) // Cada puntero ocupa 4 bytes

	// Escribir cada puntero en el slice
	for i, block := range dataBlocks {
		binary.LittleEndian.PutUint32(indice[i*4:], uint32(block))
	}

	// Escribir el contenido del índice en el bloque índice
	return escribirBloque(indexBlock, indice, nombreArchivo, "INDICE")
}

func escribirBloques(content []byte, dataBlocks []int, nombreArchivo string) error {
	for i, bloque := range dataBlocks {
		inicio := i * config.BlockSize
		fin := min(inicio+config.BlockSize, len(content))

		if err := escribirBloque(bloque, content[inicio:fin], nombreArchivo, "DATOS"); err != nil {
			return fmt.Errorf("error al escribir en bloque %d: %v", bloque, err)
		}

	}
	return nil
}

func escribirBloque(bloque int, contenido []byte, nombreArchivo string, tipoBloque string) error {
	offset := int64(bloque * config.BlockSize)
	_, err := filesystem.Bloques.WriteAt(contenido, offset)
	// slog.Debug("Se escribio en el bloque", "bloque", bloque)
	// Simulación de retardo

	slog.Info(fmt.Sprintf("## Acceso Bloque - Archivo: %s - Tipo Bloque: %s - Bloque File System %d", nombreArchivo, tipoBloque, bloque))
	time.Sleep(time.Duration(config.BlockAccessDelay) * time.Millisecond)
	return err
}

/*--------AUX---------*/
func calcularBloquesLibres(bitmap []byte) int {

	bloquesLibres := 0
	for _, byte := range bitmap {
		for bit := 0; bit < 8; bit++ {
			if byte&(1<<bit) == 0 {
				bloquesLibres++
			}
		}
	}
	return bloquesLibres
}
