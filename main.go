package main

import (
	"bufio"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Conn struct {
	connections map[net.Conn]*bufio.ReadWriter
}

func NewConn() *Conn {
	return &Conn{
		connections: make(map[net.Conn]*bufio.ReadWriter),
	}
}

func main() {

	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
	wskey, exists := os.LookupEnv("WS_KEY")
	if exists {
		fmt.Println(wskey)
	}

	guid := generateGUID()

	connections := NewConn()

	http.Handle("/", http.FileServer(http.Dir("static")))
	fmt.Println("Server is running on :3000")
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		noWsHandler(w, r, guid, connections)
	})
	http.ListenAndServe(":3000", nil)

}

func noWsHandler(w http.ResponseWriter, req *http.Request, guid string, conn *Conn) {

	if req.Header.Get("Upgrade") != "websocket" {
		return
	}
	if req.Header.Get("Connection") != "Upgrade" {
		return
	}

	fmt.Println(w, "New connnection")

	// fmt.Println(w.Header())
	// fmt.Println(req.Header)

	// log.Println("guid: ", guid)

	secKey := req.Header.Get("Sec-Websocket-Key")
	if secKey == "" {
		return
	}
	// log.Println("secKey: ", secKey)

	sum := secKey + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	//					   "2A24432C-3620-A604-883B-F4F5E9237DEF"
	// log.Println("sum: ", sum)

	hash := sha1.Sum([]byte(sum))
	// log.Println("hash: ", hash)

	str := base64.StdEncoding.EncodeToString(hash[:])
	// log.Println("str: ", str)

	rw_conn, rwbuf, err := w.(http.Hijacker).Hijack()

	conn.connections[rw_conn] = rwbuf
	if err != nil {
		panic("Hijack failed: " + err.Error())
	}
	defer rw_conn.Close()

	if err := handShake(rwbuf, req, str); err != nil {
		panic("handShake failed: " + err.Error())
	}

	// buf := make([]byte, 1024)
	for {

		// rwbuf.Write([]byte("mes fromm rwbuf"))

		// n, err := rwbuf.Read(buf)
		// if err != nil {
		// 	return
		// }
		// fmt.Println(buf[:n])
		mes, _ := ReadMessage(rwbuf)
		// log.Println("DecodeWsMessage: ", string(mes))
		// if err != nil {
		// 	panic("decodeWSMessage failed: " + err.Error())
		// }
		// // broadcast(mes, conn)
		// for client, b := range conn.connections {
		// 	_, err := b.Write(mes)
		// 	log.Println("mes: ", string(mes))
		// 	if err != nil {
		// 		fmt.Println("Error writing to a client:", err)
		// 		// Optionally, you can remove the problematic connection from the map.
		// 		delete(conn.connections, client)
		// 	}
		// }

		// broadcast(mes, conn)
		// writeToClient(w, rwbuf, mes)
		_, err := WriteMessage(rwbuf, 1, mes)
		if err != nil {
			panic("WriteWebSocketMessage failed: " + err.Error())
		}

	}

}

func broadcast(b []byte, cs *Conn) {
	for _, buf := range cs.connections {
		go func(buf *bufio.ReadWriter) {
			log.Println("bbbb: ", string(b))
			if _, err := buf.Write(b); err != nil {
				fmt.Println("Errorrr: ", err)
			}
		}(buf)
	}
}

// func broadcast(b []byte, conn *Conn) {
// 	for client, _ := range conn.connections {
// 		_, err := client.Write(b)
// 		if err != nil {
// 			fmt.Println("Error writing to a client:", err)
// 			// Optionally, you can remove the problematic connection from the map.
// 			delete(conn.connections, client)
// 		}
// 	}
// }

func generateGUID() (guid string) {
	key := make([]byte, 16)
	// n, _ := io.ReadFull(rand.Reader, key)
	_, err := rand.Read(key)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}
	guid = fmt.Sprintf("%X-%X-%X-%X-%X", key[0:4], key[4:6], key[6:8], key[8:10], key[10:])
	// fmt.Println(guid)
	return
}

func handShake(rw *bufio.ReadWriter, req *http.Request, secureStr string) error {
	var scheme string
	if req.TLS != nil {
		scheme = "wss"
	} else {
		scheme = "ws"
	}
	rw.WriteString("HTTP/1.1 101 Switching Protocols\r\n")
	location, _ := url.ParseRequestURI(scheme + "://" + req.Host + req.URL.RequestURI())
	rw.WriteString("GET " + location.RequestURI() + " HTTP/1.1\r\n")

	rw.WriteString("Host: " + removeZone(location.Host) + "\r\n")
	rw.WriteString("Upgrade: websocket\r\n")
	rw.WriteString("Connection: Upgrade\r\n")

	rw.WriteString("Origin: " + strings.ToLower(getOrigin(req)) + "\r\n")
	rw.WriteString("Sec-Websocket-Accept: " + secureStr + "\r\n\r\n")
	// fmt.Println("Sec-Websocket-Accept: ", secureStr)

	if err := rw.Flush(); err != nil {
		return err
	}
	return nil
}

func removeZone(host string) string {
	if !strings.HasPrefix(host, "[") {
		return host
	}
	i := strings.LastIndex(host, "]")
	if i < 0 {
		return host
	}
	j := strings.LastIndex(host[:i], "%")
	if j < 0 {
		return host
	}
	return host[:j] + host[i:]
}

func getOrigin(req *http.Request) string {
	origin := req.Header.Get("Origin")
	return origin
}
