package main

import (
	"net/http"
	"os"

	"github.com/sisoputnfrba/tp-golang/utils"
)

type Config struct {
	Port             int    `json:"port"`
	IPMemory         string `json:"ip_memory"`
	PortMemory       int    `json:"port_memory"`
	MountDir         string `json:"mount_dir"`
	BlockSize        int    `json:"block_size"`
	BlockCount       int    `json:"block_count"`
	BlockAccessDelay int    `json:"block_access_delay"`
	LogLevel         string `json:"log_level"`
}

type DumpFile struct {
	FileName    string
	Size        int
	Content     []byte
	IndexBlocks []int
}

type BodyDumpFile struct {
	FileName string
	Size     int
	Content  []byte
}
type FileSystem struct {
	Bitmap   []byte                   // Se carga con el archivo bitmap.dat
	Bloques  *os.File                 // Se carga con el archivo bloques.dat
	Metadata map[string]*FileMetadata // Mapa que vincula el nombre de un archivo con su metadata
}
type FileMetadata struct {
	IndexBlock int
	Size       int
}

var rutas = map[string]http.HandlerFunc{
	"/handshake/{modulo}":     utils.IniciarServer,
	"/filesystem/dump_memory": CrearArchivoDump,
}

var config Config
var filesystem *FileSystem
