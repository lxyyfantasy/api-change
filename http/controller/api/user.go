package api

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/model"
	requstform "github.com/lejianwen/rustdesk-api/v2/http/request/api"
	apiResp "github.com/lejianwen/rustdesk-api/v2/http/response/api"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"github.com/lejianwen/rustdesk-api/v2/utils"
	"net/http"
)

type User struct {
}

// Info 用户信息
// @Tags 用户
// @Summary 用户信息
// @Description 用户信息
// @Accept  json
// @Produce  json
// @Success 200 {object} apiResp.UserPayload
// @Failure 500 {object} response.Response
// @Router /currentUser [get]
// @Security token
func (u *User) Info(c *gin.Context) {
	user := service.AllService.UserService.CurUser(c)
	up := (&apiResp.UserPayload{}).FromUser(user)
	c.JSON(http.StatusOK, up)
}

// accountResp 客户端 AccountApi 约定返回 {code, msg}，code == "OK" 为成功
func accountResp(c *gin.Context, code, msg string) {
	c.JSON(http.StatusOK, gin.H{
		"code": code,
		"msg":  msg,
	})
}

// Register 客户端注册接口（公开，无 /api 前缀）
// 与网页端共用 app.register 开关和 UserService.Register 逻辑
func (u *User) Register(c *gin.Context) {
	// 与网页端共用的注册开关
	if !global.Config.App.Register {
		accountResp(c, "ERROR", "注册功能未开启")
		return
	}
	form := &requstform.RegisterForm{}
	if err := c.ShouldBindJSON(form); err != nil {
		accountResp(c, "ERROR", "参数错误: "+err.Error())
		return
	}
	// 状态取配置 register-status，和网页端创建用户行为一致
	status := model.StatusCode(global.Config.App.RegisterStatus)
	user := service.AllService.UserService.Register(form.Username, "", form.Password, status)
	if user == nil {
		accountResp(c, "ERROR", "注册失败：用户名已存在")
		return
	}
	accountResp(c, "OK", "注册成功")
}

// 改的是当前登录账号自己的密码，不是 RustDesk ID 密码
func (u *User) ChangePassword(c *gin.Context) {
	form := &requstform.ChangePasswordForm{}
	if err := c.ShouldBindJSON(form); err != nil {
		accountResp(c, "ERROR", "参数错误: "+err.Error())
		return
	}
	cur := service.AllService.UserService.InfoByUsername(form.Username)
	if cur == nil || cur.Id == 0 {
		accountResp(c, "ERROR", "用户不存在")
		return
	}
	ok, _, err := utils.VerifyPassword(cur.Password, form.OldPassword)
	if err != nil || !ok {
		accountResp(c, "ERROR", "原密码错误")
		return
	}
	if err := service.AllService.UserService.UpdatePassword(cur, form.NewPassword); err != nil {
		accountResp(c, "ERROR", "密码修改失败: "+err.Error())
		return
	}
	accountResp(c, "OK", "密码修改成功")
}