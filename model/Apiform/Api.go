package Apiform

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bingoohuang/sshman/common"
	"github.com/bingoohuang/sshman/config"
	"github.com/bingoohuang/sshman/model"
	"github.com/garyburd/redigo/redis"
	"github.com/satori/go.uuid"
	"log"
)

type Login struct {
	Phone int    `form:"phone" binding:"required"`
	Code  string `form:"code" binding:"required"`
}

type Send struct {
	Phone string `form:"phone" binding:"required"`
}

type Slist struct {
	Page  int `form:"page" binding:"required"`
	Limit int `form:"limit" binding:"required"`
}

type List_resp struct {
	List  []model.Server
	Count uint
}

type GetTerm struct {
	ID       uint   `form:"id" binding:"required"`
	Password string `form:"setpass" binding:"required"`
}

type WsAuth struct {
	Sid string `uri:"sid" binding:"required,uuid"`
}

type Edit struct {
	ID       uint   `form:"id" binding:"required"`
	Nickname string `form:"nickname"`
	Ip       string `form:"ip"`
	Port     int    `form:"port"`
	Username string `form:"username"`
}

type EditPass struct {
	ID       uint   `form:"id" binding:"required"`
	Password string `form:"password" binding:"required"`
}

type Addser struct {
	Nickname string `form:"nickname"`
	Ip       string `form:"ip" binding:"required"`
	Port     int    `form:"port" binding:"required"`
	Username string `form:"username" binding:"required"`
	Password string `form:"password" binding:"required"`
}

type SerInfo struct {
	ID       uint
	Ip       string
	Port     int
	Username string
	Password string
	BindUser uint
}

func (s *Send) SendCaptcha(ip string) (err error) {
	cache := config.Cache.Get()
	defer cache.Close()
	ipExists, _ := redis.Bool(cache.Do("EXISTS", ip))
	phoneExists, _ := redis.Bool(cache.Do("EXISTS", s.Phone+"_time"))
	if ipExists || phoneExists {
		err = errors.New("请勿频繁发送验证码")
		return
	}
	cache.Send("MULTI")                             //开启事务操作
	cache.Send("SETEX", s.Phone+"_time", 60*2, nil) //记录手机号与IP，防止重复发送
	cache.Send("SETEX", ip, 60*2, nil)
	capcha, err := common.Sendsms(s.Phone)
	if err != nil {
		cache.Do("DISCARD") //发送失败则取消事务
		log.Println(err.Error())
		return
	}
	cache.Send("SETEX", s.Phone, 60*5, capcha) //延长过期时间，用于校验
	cache.Do("EXEC")                           //提交事务
	return
}

func (l *Login) Verify() (key, code string) {
	return fmt.Sprintf("%d", l.Phone), l.Code
}

func (t *GetTerm) Decode(server model.Server) (sid string, err error) {
	sid = uuid.Must(uuid.NewV4(), nil).String()
	sPass, err := common.AesDecryptCBC(server.Password, []byte(t.Password))
	if err != nil {
		return "", err
	}
	if sPass == "" {
		return "", errors.New("秘钥验证失败")
	}

	serInfo := SerInfo{
		ID:       server.ID,
		Ip:       server.Ip,
		Port:     server.Port,
		Username: server.Username,
		Password: sPass,
		BindUser: server.BindUser,
	}

	sInfo, _ := json.Marshal(serInfo)

	cache := config.Cache.Get()
	defer cache.Close()
	cache.Do("SETEX", sid, 10, sInfo) //缓存10s，用于建立连接和验证权限

	return sid, nil
}
