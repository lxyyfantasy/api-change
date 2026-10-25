package admin

import (
    "net/http"
    "strconv"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/lejianwen/rustdesk-api/v2/model"
    "github.com/lejianwen/rustdesk-api/v2/service"
)

type Activation struct {
}

func (ct *Activation) PeerAll(c *gin.Context) {
    list := make([]*model.Peer, 0)
    service.DB.Find(&list)
    c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": list}})
}

func (ct *Activation) ServerConfigGet(c *gin.Context) {
    sc := &model.ServerConfig{}
    service.DB.Where("id = 1").First(sc)
    c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
        "server_key": sc.ServerKey,
        "ip1":        sc.Ip1,
        "ip2":        sc.Ip2,
    }})
}

func (ct *Activation) ServerConfigPut(c *gin.Context) {
    f := &struct {
        Key      string `json:"key"`
        Ip1      string `json:"ip1"`
        Ip2      string `json:"ip2"`
        Password string `json:"password" binding:"required"`
    }{}
    if err := c.ShouldBindJSON(f); err != nil {
        c.JSON(http.StatusOK, gin.H{"code": 101, "message": "参数错误"})
        return
    }
    u := service.AllService.UserService.CurUser(c)
    if u == nil || !service.AllService.UserService.IsAdmin(u) {
        c.JSON(http.StatusOK, gin.H{"code": 101, "message": "无权限"})
        return
    }
    if service.AllService.UserService.InfoByUsernamePassword(u.Username, f.Password).Id == 0 {
        c.JSON(http.StatusOK, gin.H{"code": 101, "message": "密码错误"})
        return
    }
    service.DB.Model(&model.ServerConfig{}).Where("id = 1").Updates(map[string]interface{}{
        "server_key": f.Key,
        "ip1":        f.Ip1,
        "ip2":        f.Ip2,
        "updated_at": time.Now().Unix(),
    })
    c.JSON(http.StatusOK, gin.H{"code": 0})
}

func (ct *Activation) TakeCodes(c *gin.Context) {
    kind := c.Query("kind")
    days, _ := strconv.Atoi(c.Query("days"))
    n, _ := strconv.Atoi(c.Query("n"))
    password := c.Query("password")
    if days <= 0 || n <= 0 {
        c.JSON(http.StatusOK, gin.H{"code": 101, "message": "参数错误"})
        return
    }
    u := service.AllService.UserService.CurUser(c)
    if u == nil || !service.AllService.UserService.IsAdmin(u) {
        c.JSON(http.StatusOK, gin.H{"code": 101, "message": "无权限"})
        return
    }
    if service.AllService.UserService.InfoByUsernamePassword(u.Username, password).Id == 0 {
        c.JSON(http.StatusOK, gin.H{"code": 101, "message": "密码错误"})
        return
    }
    codes := service.AllService.ActivationService.TakeCodes(kind, days, n)
    c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": codes}})
}

func (ct *Activation) RemovedDevices(c *gin.Context) {
    list := make([]*model.RemovedDevice, 0)
    service.DB.Order("removed_at desc").Find(&list)
    c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": list}})
}

func (ct *Activation) RemovedAccounts(c *gin.Context) {
    list := make([]*model.EmptyAccount, 0)
    service.DB.Order("empty_since desc").Find(&list)
    c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": list}})
}

func (ct *Activation) UsedCodes(c *gin.Context) {
    kind := c.Query("kind")
    if kind == "lease" {
        list := make([]*model.LeaseCodeUsed, 0)
        service.DB.Order("used_at desc").Find(&list)
        c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": list}})
        return
    }
    list := make([]*model.HostCodeUsed, 0)
    service.DB.Order("used_at desc").Find(&list)
    c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": list}})
}