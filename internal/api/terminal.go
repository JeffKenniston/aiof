package api

import (
	"log/slog"
	"net/http"
	"os"
	"os/exec"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (s *Server) handleTerminal(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("WS upgrade failed", slog.Any("error", err))
		return
	}
	defer ws.Close()

	dir := r.URL.Query().Get("project")
	if dir == "" {
		s.projectMu.Lock()
		dir = s.activeProjectDir
		s.projectMu.Unlock()
	}
	
	c := exec.Command("bash")
	c.Dir = dir
	c.Env = append(os.Environ(), "TERM=xterm")
	ptmx, err := pty.Start(c)
	if err != nil {
		s.logger.Error("PTY start failed", slog.Any("error", err))
		return
	}
	defer ptmx.Close()

	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				break
			}
			if err := ws.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
				break
			}
		}
	}()
	go func() {
		for {
			_, msg, err := ws.ReadMessage()
			if err != nil {
				break
			}
			ptmx.Write(msg)
		}
	}()
	
	c.Wait()
}
