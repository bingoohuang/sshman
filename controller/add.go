package controller

import (
	"github.com/bingoohuang/sshman/common"
	"github.com/bingoohuang/sshman/config"
	"github.com/bingoohuang/sshman/model"
	"github.com/bingoohuang/sshman/model/Apiform"
	"github.com/gin-gonic/gin"
)

func Addser(c *gin.Context) {
	var resp Apiform.Resp
	newToken := c.MustGet("token").(string)
	if newToken != "" { //更新Token逻辑
		resp.Token = newToken
	}
	uid := c.MustGet("uid").(uint)
	var info Apiform.Addser
	resp.Code = config.C_from_err
	resp.Msg = "数据错误"
	if c.ShouldBind(&info) == nil {
		if common.CheckIp(info.Ip) {
			db := config.DB()
			defer db.Close()
			result := db.Create(&model.Server{
				Ip:       info.Ip,
				Port:     info.Port,
				Username: info.Username,
				Password: info.Password,
				Nickname: info.Nickname,
				BindUser: uid,
			})
			if result.RowsAffected == 1 && result.Error == nil {
				resp.Code = config.C_nil_err
				resp.Msg = "保存成功"
			} else {
				resp.Code = config.S_Db_err
				resp.Msg = "保存失败"
			}
		}
	}
	c.JSON(200, resp)
}
