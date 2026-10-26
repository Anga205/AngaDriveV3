// Package requestHandler implements the AngaDrive websocket transport: it
// upgrades and tracks connections, decodes incoming messages, and delegates to
// the service layer. Business logic lives in package service; broadcasting
// lives in package hub.
package requestHandler

import (
	"angadrive/config"
	"angadrive/globals"
	"angadrive/hub"
	"angadrive/info"
	"angadrive/runners"
	"angadrive/service"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// SetupWebsocket registers the /ws route and starts the background broadcast
// loops. It must be called once at startup.
func SetupWebsocket(r *gin.Engine) {
	runners.Init(runnerNotifier{})
	info.InitializeSysInfo()
	hub.InitCounters()
	hub.StartTickers()

	r.GET("/ws", func(c *gin.Context) {
		if c.Request.Host != config.WebURL {
			c.JSON(http.StatusForbidden, gin.H{"error": "Websocket connection not allowed from this host"})
			return
		}
		conn, err := upgradeToWebSocket(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upgrade to websocket"})
			return
		}
		defer conn.Close()

		hub.Register(conn)
		defer hub.Unregister(conn)

		done := make(chan bool)
		go reader(conn, done)
		<-done
	})
}

func upgradeToWebSocket(c *gin.Context) (*websocket.Conn, error) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	return upgrader.Upgrade(c.Writer, c.Request, nil)
}

func reader(conn *websocket.Conn, done chan bool) {
	defer func() { done <- true }()
	for {
		messageType, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				fmt.Printf("Error reading message: %v\n", err)
			}
			break
		}
		if messageType != websocket.TextMessage {
			continue
		}
		var message globals.IncomingMessage
		if err := json.Unmarshal(msg, &message); err != nil {
			fmt.Printf("[%s] Error unmarshalling JSON: %v\n", time.Now().Format("03:04:05 PM, 02 Jan 2006"), err)
			continue
		}
		rawData, _ := json.Marshal(message.Data)
		dispatchMessage(conn, message.Type, rawData)
	}
}

// updateConnAuth refreshes the session credentials for a connection after a
// successful authenticated request.
func updateConnAuth(conn *websocket.Conn, req authInfo) {
	hub.Update(conn, func(d *globals.WebsocketData) {
		if req.Email != "" && req.Password != "" {
			account, err := service.AccountForCredentials(req.Email, req.Password)
			if err != nil {
				return
			}
			d.UserInfo.Token = ""
			d.UserInfo.Email = account.Email
			d.UserInfo.HashedPassword = account.HashedPassword
			return
		}
		d.UserInfo.Token = req.Token
		d.UserInfo.Email = ""
		d.UserInfo.HashedPassword = ""
	})
}
