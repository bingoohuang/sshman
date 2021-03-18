package controller

import (
	"github.com/bingoohuang/sshman/common"
	"github.com/bingoohuang/sshman/config"
	"github.com/bingoohuang/sshman/model/apiform"
	"github.com/gin-gonic/gin"
)

func Send(c *gin.Context) {
	var resp apiform.Resp
	var send apiform.Send
	resp.Code = config.C_phone_err
	resp.Msg = "手机号未提交！"
	if c.ShouldBind(&send) == nil {
		if common.VerifyMobileFormat(send.Phone) {
			if err := send.SendCaptcha(c.ClientIP()); err != nil {
				resp.Code = config.S_send_err
				resp.Msg = err.Error()
			} else {
				resp.Code = config.C_nil_err
				resp.Msg = "发送成功！"
			}
		} else {
			resp.Code = config.C_phone_err
			resp.Msg = "手机号验证失败！"
		}
	}
	c.JSON(200, resp)
}
