package common

import (
	"encoding/base64"
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/pkg/sftp"
	"log"
	"path"
	"sync"
	"time"
)

const (
	getpwd = "getpwd"
	upload = "upload"
)

type sftpReq struct {
	Type     string `json:"type"`
	FilePath string `json:"filepath"`
	FileName string `json:"filename"`
	FileData string `json:"filedata"`
}

type sftpRsp struct {
	Code int    `json:"code"`
	Type string `json:"type"`
	Msg  string `json:"msg"`
	Data string `json:"data"`
}

type SftpClient struct {
	Uid  uint
	Sftp *sftp.Client
}

type clients struct {
	*sync.RWMutex
	C map[string]*SftpClient
}

var Client clients

func init() {
	Client = clients{RWMutex: new(sync.RWMutex), C: make(map[string]*SftpClient)}
}

func (c *SftpClient) ReceiveWsMsg(wsConn *websocket.Conn, exitCh chan bool) {
	defer setQuit(exitCh)
	go c.SessionWait(wsConn, exitCh)
	for {
		select {
		case <-exitCh:
			return
		default:
			_, wsData, err := wsConn.ReadMessage()
			if err != nil {
				log.Println(err.Error())
				return
			}
			msgObj := &sftpReq{}
			if err := json.Unmarshal(wsData, msgObj); err != nil {
				log.Println("unmarshal websocket message failed:", string(wsData))
				continue
			}

			switch msgObj.Type {
			case getpwd:
				if !doGetcwd(wsConn, c) {
					return
				}
			case upload:
				if !doUpload(wsConn, c, msgObj) {
					return
				}
			}
		}
	}
}

func doUpload(wsConn *websocket.Conn, c *SftpClient, msgObj *sftpReq) bool {
	rsp := sftpRsp{}
	rsp.Code = 200
	rsp.Type = "upload"
	uploadData, err := base64.StdEncoding.DecodeString(msgObj.FileData)
	if err != nil {
		rsp.Code = 401
		rsp.Msg = "文件解析失败"
		rsp.Data = err.Error()
		log.Println("sftp base64decode err:", err.Error())
		msg, _ := json.Marshal(rsp)
		if err := wsConn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Println("sftp base64decode send err:", err.Error())

		}
		return false
	}

	if err := c.Sftp.MkdirAll(msgObj.FilePath); err != nil {
		rsp.Code = 402
		rsp.Msg = "服务器创建目录失败"
		rsp.Data = err.Error()
		log.Println("sftp mkdir err:", err.Error())
		msg, _ := json.Marshal(rsp)
		if err := wsConn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Println("sftp mkdir send err:", err.Error())
		}
		return false
	}

	filepath := path.Join(msgObj.FilePath, msgObj.FileName)
	file, err := c.Sftp.Create(filepath)
	if err != nil {
		rsp.Code = 403
		rsp.Msg = "服务器文件创建失败"
		rsp.Data = err.Error()
		log.Println("sftp create file err:", err.Error())
		msg, _ := json.Marshal(rsp)
		if err := wsConn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Println("sftp create file send err:", err.Error())
		}
		return false
	}

	defer file.Close()

	if _, err := file.Write(uploadData); err != nil {
		rsp.Code = 405
		rsp.Msg = "服务器文件写入失败"
		rsp.Data = err.Error()
		log.Println("sftp write file err:", err.Error())
		msg, _ := json.Marshal(rsp)
		if err := wsConn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Println("sftp write file send err:", err.Error())
		}
		return false
	}
	rsp.Code = 200
	rsp.Msg = "OK"
	rsp.Data = filepath
	msg, _ := json.Marshal(rsp)
	if err := wsConn.WriteMessage(websocket.TextMessage, msg); err != nil {
		log.Println("sftp write file send err:", err.Error())
		return false
	}

	return true
}

func doGetcwd(wsConn *websocket.Conn, c *SftpClient) bool {
	rsp := sftpRsp{}
	rsp.Code = 200
	rsp.Type = "pwd"
	wd, err := c.Sftp.Getwd()
	if err != nil {
		rsp.Code = 404
		rsp.Msg = "服务器Path获取失败"
		rsp.Data = err.Error()
		log.Println("sftp getpwd err:", err.Error())
	}
	rsp.Data = wd
	msg, _ := json.Marshal(rsp)
	if err := wsConn.WriteMessage(websocket.TextMessage, msg); err != nil {
		log.Println("sftp client getpwd err:", err.Error())
		return false
	}

	return true
}

func (c *SftpClient) SessionWait(wsConn *websocket.Conn, quitChan chan bool) {
	timer := time.NewTicker(time.Second * 30)
	defer timer.Stop()
	defer setQuit(quitChan)
	for {
		select {
		case <-timer.C:
			{
				if err := wsConn.WriteMessage(websocket.TextMessage, []byte("pong")); err != nil {
					log.Println("sftp pong send err :", err.Error())
					return
				}
			}
		case <-quitChan:
			return
		}
	}
}

func setQuit(ch chan bool) {
	ch <- true
}
