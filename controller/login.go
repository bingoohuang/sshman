package controller

import (
	"github.com/bingoohuang/sshman/common"
	"github.com/bingoohuang/sshman/config"
	"github.com/bingoohuang/sshman/model"
	"github.com/bingoohuang/sshman/model/apiform"
	"github.com/gin-gonic/gin"
)

func Login(c *gin.Context) {
	var resp apiform.Resp
	resp.Code = config.C_from_err
	resp.Msg = "手机号和验证码不能为空！"
	var user apiform.Login
	if c.ShouldBind(&user) == nil {
		if common.Verify(&user) {
			var userinfo model.User
			db := config.DB()
			defer db.Close()
			db.Where(model.User{Phone: user.Phone}).FirstOrCreate(&userinfo)
			newToken, err := common.ReleaseToken(userinfo.ID)
			if err == nil && userinfo.ID > 0 {
				resp.Code = config.C_nil_err
				resp.Msg = "登陆成功"
				resp.Data = userinfo
				resp.Token = newToken
			} else {
				resp.Code = config.S_auth_err
				resp.Msg = "Token创建失败"
			}
		} else {
			resp.Code = config.S_Verify_err
			resp.Msg = "验证码校验失败"
		}
	}
	//log.Printf(c.ClientIP())
	c.JSON(200, resp)
}
