# Webtail

Webtail is a utility tool that executes a specified command, captures its `stdout` and `stderr` in real-time, and streams the log output to clients via WebSocket.

## Features

- Execute any command and capture its outputs.
- Real-time streaming of log outputs over WebSocket.
- Set custom working directory and listening port for flexibility.

## Installation

`clone` & `go build`

## Usage

To use Webtail, execute it with the desired options followed by the command you wish to run:

```
webtail.exe [options] <command> [command arguments...]
```

### Options

- `-cwd <path>`: Set the working directory for the command.
- `-port <number>`: Set the port number for the WebSocket server.
- `-interface <address>`: Set the interface address for the WebSocket server.

### Example

```bash
webtail.exe -cwd C:\src\ -port 9999 ping google.com -t
```

In the above example:

- The command `ping google.com -t` is executed.
- The working directory for the command is set to `C:\src\`.
- The WebSocket server listens on port 9999.
- You can open http://localhost:9999/ in a browser to view the real-time logs.
- WebSocket Clients can connect to `ws://localhost:9999/logs` to view the real-time logs.
