package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"time"
)

const direccion = "127.0.0.1:1234"

var nombre string

func main() {
	var opcion uint
	var mensaje string
	var in *bufio.Reader
	conn, err := net.Dial("tcp", direccion) //Conexion
	defer conn.Close()
	if err != nil {
		log.Fatal("Error de conexion")
		os.Exit(1)
	}
	fmt.Println("Escriba su nombre: ")
	fmt.Scanln(&nombre)
	//Se envia la cadena(en forma de bytes)con la opcion 5 para establecer el nombre del usuario
	EnviarBytesAConn(conn, []byte("5 "+nombre))
	go SalidaDelServidor(conn) // Se recibe la respuesta del servidor de manera concurrente para distribuirla a los clientes
	time.Sleep(time.Second * 1)
	for {
		fmt.Println("1) Enviar mensaje")
		fmt.Println("2) Subir archivos")
		fmt.Println("3) Mostrar archivos")
		fmt.Println("4) Descargar archivos")
		fmt.Println("0) Salír")
		fmt.Scan(&opcion)
		switch opcion {
		case 1: // Opcion 1. Escribir mensaje
			fmt.Println("Escriba el mensaje: ")
			//Se crea el buffer
			for mensaje == "" {
				in = bufio.NewReader(os.Stdin)
				buf, _, _ := in.ReadLine()
				mensaje = string(buf)
			}
			mensaje = "1 " + mensaje                // El servidor leera "1" y sabra que es un mensaje
			EnviarBytesAConn(conn, []byte(mensaje)) //Se envia a la conexion el mensaje en Bytes
			time.Sleep(time.Second * 1)
			mensaje = ""
		case 2: //Opcion 2. Subir archivo
			fmt.Println("Escriba el nombre del archivo: ")
			//se crea el buffer
			for mensaje == "" {
				in = bufio.NewReader(os.Stdin)
				buf, _, _ := in.ReadLine()
				mensaje = string(buf)
			}
			mensaje = "2 " + mensaje // El servidor leera "2" y sabra que es un mensaje
			SubirArchivo(conn, mensaje)
			time.Sleep(time.Second * 1)
			mensaje = ""
		case 3: //Opcion 3. Mostrar archivos
			mensaje = "3"
			EnviarBytesAConn(conn, []byte(mensaje))
			time.Sleep(time.Second * 1)
			mensaje = ""
		case 4: //Opcion 4. Descargar archivos
			fmt.Println("Escriba el nombre del archivo: ")
			for mensaje == "" {
				in = bufio.NewReader(os.Stdin)
				buf, _, _ := in.ReadLine()
				mensaje = string(buf)
			}
			mensaje = "4 " + mensaje
			EnviarBytesAConn(conn, []byte(mensaje))
			time.Sleep(time.Second * 1)
			mensaje = ""
		case 0: //Opcion 0. Salír
			return
		default:
			fmt.Println("Opcion incorrecta")
		}
	}
}

// SubirArchivo envia el archivo al servidor
func SubirArchivo(conn net.Conn, respuesta string) {
	EnviarBytesAConn(conn, []byte(respuesta))
	nameArch := respuesta[2:]                   //A partir de la posicion 2 contiene el nombre del archivo
	archBytes, err := ioutil.ReadFile(nameArch) //Se lee el archivo y se convierte en bytes
	if err != nil {
		log.Fatal(err)
	}
	archtam := len(archBytes)
	preSend := CombinarBytes(IntABytes(archtam), archBytes) //Se combina el archivo en bytes con su tamaño
	_, err = conn.Write(preSend)                            //Se esctribe en la conexion el archivo con su tamaño en bytes
	if err != nil {
		log.Fatal(err)
	}
}

//DescargarArchivo descarga el archivo(para el cliente en uso) que esta en el servidor
func DescargarArchivo(respuesta string, conn net.Conn) {
	nameArch := respuesta[2:]
	fmt.Println("descargando el archivo " + nameArch + " ...")
	dirArch := "./descargas/" + nameArch
	//Si la carpeta no existe entonces se crea
	if !Existe("./descargas") {
		err := os.Mkdir("descargas", os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}
	}
	archBytes := RecibirBytesDesdeConn(conn) // Se recibe el archivo en bytes desde el servidor
	if len(archBytes) == 5 && string(archBytes) == "error" {
		fmt.Println("El archivo " + nameArch + " No existe. Intente escribiendo otro")
	} else {
		//Si el archivo ya existe en el directorio se sobreescribe
		if Existe(dirArch) {
			fmt.Println("El archivo " + nameArch + " ya existe, se va a sobreescribir")
			err := os.Remove(dirArch)
			if err != nil {
				log.Fatal(err)
			}
		}
		newArch, err := os.Create(dirArch) //Se crea el archivo (en blanco) en la carpeta "descargas"
		if err != nil {
			log.Fatal(err)
		}
		_, err = newArch.Write(archBytes)
		if err != nil {
			log.Fatal(err)
		} else {
			fmt.Println("El archivo " + nameArch + " fue descargado con exito")
		}
		newArch.Close()
	}
}

//SalidaDelServidor Recibe los mensajes del servidor y los distribuye a los clientes
func SalidaDelServidor(conn net.Conn) {
	for {
		respuestaBytes := RecibirBytesDesdeConn(conn) //Retorna la respuesta del servidor en bytes
		respuesta := string(respuestaBytes)
		if len(respuesta) < 1 {
			fmt.Println(respuesta)
		} else {
			if respuesta[:1] == "4" { //Si el primer caracter de la cadena es igual a 4 se activa la opcion Descargar
				DescargarArchivo(respuesta, conn)
			} else {
				fmt.Println(respuesta)
			}
		}
	}
}

//EnviarBytesAConn Cada cadena capturada en la consola se envia al servidor en forma de bytes
func EnviarBytesAConn(conn net.Conn, msgBytes []byte) {
	length := len(msgBytes)
	msgSend := CombinarBytes(IntABytes(length), msgBytes)
	_, err := conn.Write(msgSend)
	if err != nil {
		panic(err)
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
