package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
)

const direccion = "127.0.0.1:1234"

type cliente chan<- string

var ccentrando = make(chan cliente) //Almacena al cliente que entra por primera vez al chat
var ccsaliendo = make(chan cliente) // Almacena al cliente que va a salir del chat para posteriormente eliminarlo
var cmensajes = make(chan string)   //Almacena los mensajes de los clientes

func main() {
	listener, err := net.Listen("tcp", direccion)
	if err != nil {
		log.Fatal(err)
	}
	go Transmisor() //Se controla el estado de los clientes(entrando, mandando mensaje y saliendo)
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
		go Conexion(conn)
	}
}

//Transmisor se encarga de controlar el estado de los cientes(entrando, mandando mensaje y saliendo)
func Transmisor() {
	clientes := make(map[cliente]bool) // mapa de todos los clientes conectados en el servidor
	for {
		select {
		case cli := <-ccentrando: //Cuando llega un cliente
			clientes[cli] = true
		case msg := <-cmensajes: //Cliente manda mensaje
			fmt.Println(msg)
			//Se transmite el mensaje a cada cliente
			for cli := range clientes {
				cli <- msg
			}
			//Cuando el cliente sale se elimina del mapa y se cierra el canal
		case cli := <-ccsaliendo: //Cliente sale del servidor
			delete(clientes, cli)
			close(cli)
		}
	}
}

// Conexion maneja cada conexion de cada cliente
func Conexion(conn net.Conn) {
	ch := make(chan string)   // Respuesta del servidor al cliente
	go ClienteWrite(conn, ch) //Escribe la respuesta del servidor en la conexion del cliente
	var clienteNombre string  // Nombre del cliente en el chat
	ccentrando <- ch
	for {
		cadenaBytesCliente := RecibirBytesDesdeConn(conn)
		cadena := string(cadenaBytesCliente)
		palabras := strings.Fields(cadena) //Mensaje separado por espacios
		if len(palabras) < 1 {             //Si el cliente salio sale de ciclo y se cierra la conexion
			break
		}
		switch palabras[0] {
		case "1": // El cliente envio un mensaje
			cmensajes <- clienteNombre + ": " + cadena[2:]
		case "2": // El cliente subio un archivo
			cmensajes <- clienteNombre + " subio el archivo " + palabras[1]
			Subir(palabras, conn, ch)
		case "3": //El cliente solicito mostrar los archivos del servidor
			//Si no existe la carpeta se crea una
			if !Existe("./archivo") {
				err := os.Mkdir("archivo", os.ModePerm)
				if err != nil {
					log.Fatal(err)
				}
			}
			archivos, err := ioutil.ReadDir("./archivo") // Lista de los archivos en la carpeta
			if err != nil {
				panic(err)
			}
			var direccion string
			for _, archivo := range archivos {
				direccion += archivo.Name() + "\n"
			}
			ch <- direccion
		case "4": //El cliente solicito descargar un archivo del servidor
			ch <- cadena
			Descargar(palabras, conn, ch)
		case "5": //El cliente solicito establecer su nombre de usuario
			if len(palabras) >= 2 {
				clienteNombre = palabras[1]
				cmensajes <- "Se conectó: " + clienteNombre
				ch <- "Bienvenido \"" + clienteNombre + "\" al chat"
			} else {
				ch <- "Nombre de usuario incorrecto. Escriba otro"
			}
		}
	}
	//Cliente salio del chat
	ccsaliendo <- ch
	cmensajes <- clienteNombre + " ha salido"
	conn.Close()
}

//Subir escribe el archivo subido por el cliente en el servidor
func Subir(palabras []string, conn net.Conn, ch cliente) {
	if len(palabras) >= 2 {
		for _, archName := range palabras[1:] {
			nameAux := strings.Split(archName, "/")
			dirArch := "./archivo/" + nameAux[len(nameAux)-1] //direccion del archivo
			//Si no existe la carpeta se crea una
			if !Existe("./archivo") {
				err := os.Mkdir("archivo", os.ModePerm)
				if err != nil {
					log.Fatal(err)
				}
			}
			//Si el archivo existe entonces se sobreescribe
			if Existe(dirArch) {
				ch <- "El archivo " + nameAux[len(nameAux)-1] + " ya existe, se va a sobreescribir!"
				err := os.Remove(dirArch)
				if err != nil {
					log.Fatal(err)
				}
			}
			//Se escribe en el archivo que se acaba de crear el contenido del archivo que subio el cliente
			archBytes := RecibirBytesDesdeConn(conn)
			newFile, err := os.Create(dirArch)
			if err != nil {
				log.Fatal(err)
			}
			_, err = newFile.Write(archBytes)
			if err != nil {
				log.Fatal(err)
			} else {
				ch <- "El archivo " + archName + " fue subido con exito!"
			}
			newFile.Close()
		}
	}
}

//Descargar envia el archivo del servidor al cliente
func Descargar(palabras []string, conn net.Conn, ch cliente) {
	if len(palabras) >= 2 {
		for _, filename := range palabras[1:] {
			newFilename := "archivo/" + filename
			fileByte, err := ioutil.ReadFile(newFilename)
			if err != nil {
				ch <- "error" //El archivo no existe
				continue
			}
			fileByteLen := len(fileByte)
			preSend := CombinarBytes(IntABytes(fileByteLen), fileByte)
			_, err = conn.Write(preSend)
			if err != nil {
				log.Fatal(err)
			}
		}
	} else {
		ch <- "La descarga ha fallado, intente con otro nombre"
	}
}

//ClienteWrite escribe datos en la conexion del cliente
func ClienteWrite(conn net.Conn, ch <-chan string) {
	for msg := range ch {
		msgByte := []byte(msg)
		msgByteNew := CombinarBytes(IntABytes(len(msgByte)), msgByte)
		conn.Write(msgByteNew)
	}
}

//RecibirBytesDesdeConn lee del servidor la conexion y la retorna
func RecibirBytesDesdeConn(conn net.Conn) []byte {
	tamMensaje := make([]byte, 4) //El tamaño maximo del mensaje esta representado en 4 bytes osea 4,294,967,296 caracteres
	conn.Read(tamMensaje)
	respuestaBytes := make([]byte, BytesAInt(tamMensaje))
	conn.Read(respuestaBytes)
	return respuestaBytes
}

//Existe Comprueba si existe el archivo o carpeta en la direccion
func Existe(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

//CombinarBytes combina el tamaño del buffer(bytes) con el buffer(bytes)
func CombinarBytes(pBytes ...[]byte) []byte {
	return bytes.Join(pBytes, []byte(""))
}

//IntABytes convierte un entero (Generalmente la longitud de un buffer) a bytes usando el formato Big Endian
func IntABytes(n int) []byte {
	x := int32(n)
	bufferBytes := bytes.NewBuffer([]byte{})
	binary.Write(bufferBytes, binary.BigEndian, x)
	return bufferBytes.Bytes()
}

//BytesAInt convierte un array de bytes a int usando el formato BigEndian
func BytesAInt(b []byte) int {
	bytesBuffer := bytes.NewBuffer(b)
	var x int32
	binary.Read(bytesBuffer, binary.BigEndian, &x)
	return int(x)
}
