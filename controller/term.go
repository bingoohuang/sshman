package controller

import (
	"encoding/json"
	"fmt"
	"github.com/bingoohuang/sshman/common"
	"github.com/bingoohuang/sshman/common/core"
	"github.com/bingoohuang/sshman/config"
	"github.com/bingoohuang/sshman/model/apiform"
	"github.com/garyburd/redigo/redis"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"strconv"
)

var upGrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024 * 1024 * 10,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type AuthMsg struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

// handle webSocket connection.
// first,we establish a ssh connection to ssh server when a webSocket comes;
// then we deliver ssh data via ssh connection between browser and ssh server.
// That is, read webSocket data from browser (e.g. 'ls' command) and send data to ssh server via ssh connection;
// the other hand, read returned ssh data from ssh server and write back to browser via webSocket API.
func WsSsh(c *gin.Context) {
	wsConn, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	if core.HandleError(c, err) {
		return
	}
	defer wsConn.Close()

	cols, err := strconv.Atoi(c.DefaultQuery("cols", "120"))
	if core.WshandleError(wsConn, err) {
		return
	}
	rows, err := strconv.Atoi(c.DefaultQuery("rows", "32"))
	if core.WshandleError(wsConn, err) {
		return
	}

	var serInfo apiform.SerInfo //接收反序列化数据
	var auth apiform.WsAuth

	if c.ShouldBindUri(&auth) != nil {
		wsConn.WriteMessage(websocket.TextMessage, []byte("参数错误\r\n"))
		wsConn.Close()
		return
	}

	for {
		_, wsData, err := wsConn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			wsConn.Close()
			return
		}
		msgObj := AuthMsg{}
		if err := json.Unmarshal(wsData, &msgObj); err != nil {
			log.Println("Auth : unmarshal websocket message failed:", string(wsData))
			continue
		}
		token := msgObj.Token
		claims, err := common.ParseToken(token)
		valid := claims.Valid()
		if valid != nil || err != nil {
			wsConn.WriteMessage(websocket.TextMessage, []byte("身份验证失败\r\n"))
			wsConn.Close()
			return
		}
		cache := config.Cache.Get()
		defer cache.Close()
		sInfo, err := redis.Bytes(cache.Do("GET", auth.Sid))
		if err != nil || len(sInfo) == 0 {
			wsConn.WriteMessage(websocket.TextMessage, []byte("连接超时，请重试！\r\n"))
			wsConn.Close()
			return
		}
		if json.Unmarshal(sInfo, &serInfo) != nil {
			wsConn.WriteMessage(websocket.TextMessage, []byte("服务器信息获取失败，请重试！\r\n"))
			wsConn.Close()
			return
		}
		if claims.Userid != serInfo.BindUser { //验证权限
			wsConn.WriteMessage(websocket.TextMessage, []byte("权限验证失败，请重试！\r\n"))
			wsConn.Close()
			return
		}
		break
	}
	client, err := core.NewSshClient(core.Server{
		Ip:     serInfo.Ip,
		Port:   serInfo.Port,
		User:   serInfo.Username,
		Passwd: serInfo.Password,
	})
	if core.WshandleError(wsConn, err) {
		return
	}
	defer client.Close()
	ssConn, err := core.NewSshConn(cols, rows, client) //加入sftp客户端
	if core.WshandleError(wsConn, err) {
		return
	}
	common.Client.Lock()
	common.Client.C[auth.Sid] = &common.MyClient{Uid: serInfo.BindUser, Sftp: ssConn.SftpClient}
	common.Client.Unlock()
	defer func() {
		common.Client.Lock()
		delete(common.Client.C, auth.Sid) //释放SFTP客户端
		common.Client.Unlock()
	}()
	defer ssConn.Close()
	quitChan := make(chan bool, 3)

	// most messages are ssh output, not webSocket input
	go ssConn.ReceiveWsMsg(wsConn, quitChan)
	go ssConn.SendComboOutput(wsConn, quitChan)
	go ssConn.SessionWait(quitChan)

	<-quitChan //任意协程退出则结束
	fmt.Println("Exit")
	log.Println("websocket finished")
}
