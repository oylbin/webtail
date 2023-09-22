package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

type CapturePrintWriter struct {
	source string
}

var conn *websocket.Conn
var wsMux sync.Mutex
var debug bool

// Assuming cmd is a global or accessible variable that represents the started program
var cmd *exec.Cmd

func safeWrite(data []byte) {
	wsMux.Lock()
	defer wsMux.Unlock()
	if conn != nil {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Client closed the connection: %v", err)
			} else {
				log.Printf("Failed to write message: %v", err)
			}
			conn = nil
		}
	}
}

func (w *CapturePrintWriter) Write(p []byte) (n int, err error) {

	// Decode GBK to UTF-8
	decoder := simplifiedchinese.GBK.NewDecoder()
	decodedData, _ := ioutil.ReadAll(transform.NewReader(bytes.NewReader(p), decoder))

	// Print the decoded data
	if debug {
		fmt.Printf("[%s] %s", w.source, string(decodedData))
	}
	dataWithSource := fmt.Sprintf("[%s] %s", w.source, string(decodedData))
	safeWrite([]byte(dataWithSource))
	return len(p), nil
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

const htmlContent = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Realtime Logs</title>
</head>
<body>
<div>
logs from localhost<br>
</div>
    <div id="logs"></div>
    <script>
        const logsDiv = document.getElementById('logs');
        const ws = new WebSocket('ws://localhost/logs');
        ws.onmessage = function (event) {
            const logEntry = document.createElement('div');
            logEntry.textContent = event.data;
            logsDiv.appendChild(logEntry);
        };
        ws.onclose = function () {
            console.log('WebSocket connection closed');
        };
    </script>
</body>
</html>
`

func serveHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	// get host and port that client requests
	host := r.Host
	log.Println("host: ", host)

	content := strings.Replace(htmlContent, "localhost", r.Host, -1)
	_, _ = w.Write([]byte(content))
}

func startProcess(cmdStr []string, cwd string) {
	defer os.Exit(0)
	stdoutCpw := &CapturePrintWriter{source: "stdout"}
	stderrCpw := &CapturePrintWriter{source: "stderr"}
	if len(cwd) > 0 {
		log.Println("chdir to ", cwd)
		_ = os.Chdir(cwd)
	}
	log.Println("start command: ", cmdStr)
	cmd = exec.Command(cmdStr[0], cmdStr[1:]...)
	cmd.Stdout = stdoutCpw
	cmd.Stderr = stderrCpw

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start command: %v", err)
		return
	}
	// get pid of the started process
	log.Printf("Started command with pid: %d", cmd.Process.Pid)

	if err := cmd.Wait(); err != nil {
		log.Printf("Command finished with error: %v", err)
	} else {
		log.Println("Command finished successfully")
	}
}
func handleLogs(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade to websocket: %v", err)
		return
	}
	defer c.Close()
	conn = c
	for {
		time.Sleep(time.Second * 1)
		if conn == nil {
			break
		}
		// call readMessage, it handles ping/pong internally
		messageType, message, err := c.ReadMessage()
		if err != nil {
			log.Printf("Failed to read message: %v", err)
			return
		}
		log.Printf("Received message: %d,  %s", messageType, message)
	}
	log.Println("Websocket connection closed")
}
func handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	pid := 0
	status := "stopped"
	if cmd != nil && cmd.Process != nil {
		pid = cmd.Process.Pid
		status = "running"
	}
	content := fmt.Sprintf(`{"pid": %d, "status": "%s"}`, pid, status)
	_, _ = w.Write([]byte(content))
}
func handleStop(w http.ResponseWriter, r *http.Request) {
	// return empty response with 200 status code
	w.WriteHeader(http.StatusOK)
	if cmd != nil && cmd.Process != nil {
		log.Println("stop command: ", cmd.Process.Pid)
		_ = cmd.Process.Kill()
	}
}
func startWebsocketServer(addr string) {
	http.HandleFunc("/logs", handleLogs)
	http.HandleFunc("/", serveHTML)
	http.HandleFunc("/admin/status", handleStatus)
	http.HandleFunc("/admin/stop", handleStop)
	log.Printf("listening on http://%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
func main() {
	var interfaceAddr string
	var port int
	var cwd string

	flag.StringVar(&interfaceAddr, "interface", "0.0.0.0", "Interface to bind to")
	flag.IntVar(&port, "port", 17862, "Port to listen on")
	flag.StringVar(&cwd, "cwd", "", "chdir to cwd before run command")
	flag.BoolVar(&debug, "debug", false, "Enable debug mode")
	flag.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, "Usage: %s [options] command [args...]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	commandAndArgs := flag.Args()
	log.Printf("command and args: %+q\n", commandAndArgs)
	if len(commandAndArgs) == 0 {
		_, _ = fmt.Fprintf(os.Stderr, "Command not provided. \nUsage: %s [options] command [args...]\n", os.Args[0])
		flag.PrintDefaults()
		return
	}
	go startProcess(commandAndArgs, cwd)
	addr := interfaceAddr + ":" + fmt.Sprintf("%d", port)
	go startWebsocketServer(addr)

	// Listen for termination signals
	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, syscall.SIGINT, syscall.SIGTERM)

	// Wait for termination signal
	<-terminate
	log.Println("Termination signal received")

	// Kill the started program
	if cmd != nil && cmd.Process != nil {
		log.Println("kill process")
		cmd.Process.Kill()
		// wait for process to exit, otherwise it will become zombie
		log.Println("wait for process to exit")
		cmd.Wait()
		log.Println("process exited")
	}

}
