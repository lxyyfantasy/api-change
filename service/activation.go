package service

import (
    "crypto/rand"
    "errors"
    "time"

    "github.com/lejianwen/rustdesk-api/v2/model"
)

type ActivationService struct {
}

const CodeChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func randomCode(n int) string {
    b := make([]byte, n)
    if _, err := rand.Read(b); err != nil {
        return ""
    }
    for i := range b {
        b[i] = CodeChars[int(b[i])%len(CodeChars)]
    }
    return string(b)
}

// GenerateCodes 生成激活码，不足 20 补到 20
func (as *ActivationService) GenerateCodes(kind string, days int) {
    if kind != "lease" && kind != "host" {
        return
    }
    var cnt int64
    if kind == "lease" {
        DB.Model(&model.LeaseCodeUnused{}).Where("days = ? AND expire_at = 0 AND taken_at = 0", days).Count(&cnt)
    } else {
        DB.Model(&model.HostCodeUnused{}).Where("days = ? AND expire_at = 0 AND taken_at = 0", days).Count(&cnt)
    }
    if cnt >= 20 {
        return
    }
    need := 20 - cnt
    for i := int64(0); i < need; i++ {
        for attempt := 0; attempt < 20; attempt++ {
            code := randomCode(16)
            if as.codeExists(kind, code) {
                continue
            }
            var err error
            if kind == "lease" {
                err = DB.Create(&model.LeaseCodeUnused{Code: code, Days: days}).Error
            } else {
                err = DB.Create(&model.HostCodeUnused{Code: code, Days: days}).Error
            }
            if err == nil {
                break
            }
        }
    }
}

func (as *ActivationService) codeExists(kind, code string) bool {
    var cnt int64
    if kind == "lease" {
        DB.Model(&model.LeaseCodeUnused{}).Where("code = ?", code).Count(&cnt)
        if cnt > 0 {
            return true
        }
        DB.Model(&model.LeaseCodeUsed{}).Where("code = ?", code).Count(&cnt)
        return cnt > 0
    }
    DB.Model(&model.HostCodeUnused{}).Where("code = ?", code).Count(&cnt)
    if cnt > 0 {
        return true
    }
    DB.Model(&model.HostCodeUsed{}).Where("code = ?", code).Count(&cnt)
    return cnt > 0
}

// TakeCodes 取码，标记 taken_at
func (as *ActivationService) TakeCodes(kind string, days, n int) []string {
    as.GenerateCodes(kind, days)
    var codes []string
    now := time.Now().Unix()
    if kind == "lease" {
        var list []model.LeaseCodeUnused
        DB.Where("days = ? AND taken_at = 0", days).Limit(n).Find(&list)
        for i := range list {
            DB.Model(&list[i]).Update("taken_at", now)
            codes = append(codes, list[i].Code)
        }
    } else {
        var list []model.HostCodeUnused
        DB.Where("days = ? AND taken_at = 0", days).Limit(n).Find(&list)
        for i := range list {
            DB.Model(&list[i]).Update("taken_at", now)
            codes = append(codes, list[i].Code)
        }
    }
    return codes
}

// Activate 激活
func (as *ActivationService) Activate(code, kind, deviceId, account string) (err error, expireAt int64) {
    if account == "" {
        account = "__device__"
    }
    tx := DB.Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
            err = errors.New("panic")
        }
    }()

    var days int
    var expireField string
    now := time.Now().Unix()

    switch kind {
    case "lease":
        expireField = "lease_expire_at"
        var c model.LeaseCodeUnused
        if tx.Where("code = ?", code).First(&c).Error != nil {
            var used model.LeaseCodeUsed
            if tx.Where("code = ?", code).First(&used).Error == nil && used.BindId == deviceId {
                tx.Rollback()
                return nil, 0
            }
            tx.Rollback()
            return errors.New("INVALID_CODE"), 0
        }
        days = c.Days
        tx.Delete(&c)
        tx.Create(&model.LeaseCodeUsed{Code: code, Days: days, BindId: deviceId, UsedAt: now})
    case "host":
        expireField = "host_expire_at"
        var c model.HostCodeUnused
        if tx.Where("code = ?", code).First(&c).Error != nil {
            var used model.HostCodeUsed
            if tx.Where("code = ?", code).First(&used).Error == nil && used.BindId == deviceId {
                tx.Rollback()
                return nil, 0
            }
            tx.Rollback()
            return errors.New("INVALID_CODE"), 0
        }
        days = c.Days
        tx.Delete(&c)
        tx.Create(&model.HostCodeUsed{Code: code, Days: days, BindId: deviceId, UsedAt: now})
    default:
        tx.Rollback()
        return errors.New("BAD_KIND"), 0
    }

    pe := AllService.PeerService.FindById(deviceId)
    if pe.RowId == 0 {
        pe = &model.Peer{Id: deviceId, Account: account, LastOnlineTime: time.Now().Unix()}
        tx.Create(pe)
    }
    base := pe.LeaseExpireAt
    if kind == "host" {
        base = pe.HostExpireAt
    }
    if base < now {
        base = now
    }
    expireAt = base + int64(days)*86400

    updates := map[string]interface{}{
        expireField: expireAt,
        "account":   account,
    }
    if kind == "lease" {
        updates["lease_user"] = account
        updates["lease_code"] = code
        updates["lease_days"] = days
        updates["lease_at"] = now
    } else {
        updates["host_user"] = account
        updates["host_code"] = code
        updates["host_days"] = days
        updates["host_at"] = now
    }
    tx.Model(&model.Peer{}).Where("id = ?", deviceId).Updates(updates)
    tx.Commit()
    return nil, expireAt
}

// CleanExpiredPeers 未激活或已到期满3天：移出地址簿，变游离，其他信息不动
func (as *ActivationService) CleanExpiredPeers() {
    cutoff := time.Now().Unix() - 3*86400
    cutoffTime := time.Now().Add(-3 * 24 * time.Hour)
    var list []model.Peer
    DB.Where("account != ? AND (lease_expire_at = 0 AND created_at < ? OR lease_expire_at > 0 AND lease_expire_at < ?)",
        "__device__", cutoffTime, cutoff).Find(&list)
    for _, p := range list {
        DB.Where("id = ?", p.Id).Delete(&model.AddressBook{})
        DB.Model(&model.Peer{}).Where("id = ?", p.Id).Update("account", "__device__")
    }
}

func (as *ActivationService) removePeerAndAB(id string) {
    DB.Where("id = ?", id).Delete(&model.Peer{})
    DB.Where("id = ?", id).Delete(&model.AddressBook{})
}

// CleanEmptyAccounts 名下无设备的账号：进留存表，3 天后删账号；留存表 30 天后清
func (as *ActivationService) CleanEmptyAccounts() {
    // 1. 扫普通用户，名下无设备就进留存表（已有记录不重复插）
    var users []model.User
    DB.Where("is_admin = ?", false).Find(&users)
    for _, u := range users {
        var cnt int64
        DB.Model(&model.Peer{}).Where("account = ?", u.Username).Count(&cnt)
        if cnt == 0 {
            var ea model.EmptyAccount
            if DB.Where("account = ?", u.Username).First(&ea).Error != nil {
                DB.Create(&model.EmptyAccount{Account: u.Username, EmptySince: time.Now().Unix()})
            }
        } else {
            DB.Where("account = ?", u.Username).Delete(&model.EmptyAccount{})
        }
    }

    // 2. 进留存表满 3 天的，删账号，留存记录留着
    cutoff3 := time.Now().Unix() - 3*86400
    var eas3 []model.EmptyAccount
    DB.Where("empty_since < ?", cutoff3).Find(&eas3)
    for _, ea := range eas3 {
        var u model.User
        if DB.Where("username = ?", ea.Account).First(&u).Error == nil {
            DB.Where("user_id = ?", u.Id).Delete(&model.UserToken{})
            DB.Delete(&u)
        }
    }

    // 3. 留存表记录满 30 天的，删记录
    cutoff30 := time.Now().Unix() - 30*86400
    DB.Where("empty_since < ?", cutoff30).Delete(&model.EmptyAccount{})
}

// CleanRemovedDevices 已删 ID 留存 30 天
func (as *ActivationService) CleanRemovedDevices() {
    cutoff := time.Now().Unix() - 30*86400
    DB.Where("removed_at < ?", cutoff).Delete(&model.RemovedDevice{})
}

// RefreshAccountColumn 扫地址簿填归属账号，不删行
func (as *ActivationService) RefreshAccountColumn() {
    var peers []model.Peer
    DB.Find(&peers)
    for _, p := range peers {
        var ab model.AddressBook
        var account string
        if DB.Where("id = ?", p.Id).First(&ab).Error == nil && ab.UserId > 0 {
            var u model.User
            if DB.Where("id = ?", ab.UserId).First(&u).Error == nil {
                account = u.Username
            }
        }
        if account == "" {
            account = "__device__"
        }
        DB.Model(&model.Peer{}).Where("id = ?", p.Id).Update("account", account)
    }
}

// ReportIp 上报 IP + 识别码，识别码变了则迁移设备
func (as *ActivationService) ReportIp(deviceId, ip, deviceCode string) error {
	if deviceCode != "" {
		var old model.Peer
		DB.Where("device_code = ?", deviceCode).First(&old)
		if old.RowId > 0 && old.Id != deviceId {
			var exist model.Peer
			if DB.Where("id = ?", deviceId).First(&exist).Error != nil {
				// 新 ID 不存在，复制老信息
				newPeer := &model.Peer{
					Id:             deviceId,
					Cpu:            old.Cpu,
					Hostname:       old.Hostname,
					Memory:         old.Memory,
					Os:             old.Os,
					Username:       old.Username,
					Uuid:           old.Uuid,
					Version:        old.Version,
					UserId:         old.UserId,
					GroupId:        old.GroupId,
					Alias:          old.Alias,
					Account:        old.Account,
					Category:       "被控端",
					LanIp:          ip,
					DeviceCode:     deviceCode,
					LeaseExpireAt:  old.LeaseExpireAt,
					LeaseUser:      old.LeaseUser,
					LeaseCode:      old.LeaseCode,
					LeaseDays:      old.LeaseDays,
					LeaseAt:        old.LeaseAt,
					HostExpireAt:   old.HostExpireAt,
					HostUser:       old.HostUser,
					HostCode:       old.HostCode,
					HostDays:       old.HostDays,
					HostAt:         old.HostAt,
					LastOnlineTime: time.Now().Unix(),
				}
				DB.Create(newPeer)
				// 复制老地址簿行到新 ID
				var ab model.AddressBook
				if DB.Where("id = ?", old.Id).First(&ab).Error == nil {
					newAb := ab
					newAb.RowId = 0
					newAb.Id = deviceId
					DB.Create(&newAb)
				}
				// 删老 peer + 老地址簿
				DB.Where("id = ?", old.Id).Delete(&model.Peer{})
				DB.Where("id = ?", old.Id).Delete(&model.AddressBook{})
				return nil
			}
		}
	}
	return DB.Model(&model.Peer{}).Where("id = ?", deviceId).Updates(map[string]interface{}{
		"lan_ip":           ip,
		"device_code":      deviceCode,
		"last_online_time": time.Now().Unix(),
		"category":         "被控端",
	}).Error
}

// CleanUnused 未使用 15 天清理
func (as *ActivationService) CleanUnused() {
    cutoff := time.Now().Unix() - 15*86400
    DB.Where("taken_at > 0 AND taken_at < ?", cutoff).Delete(&model.LeaseCodeUnused{})
    DB.Where("taken_at > 0 AND taken_at < ?", cutoff).Delete(&model.HostCodeUnused{})
}

// CleanUsed 已使用 30 天清理
func (as *ActivationService) CleanUsed() {
    cutoff := time.Now().Unix() - 30*86400
    DB.Where("used_at < ?", cutoff).Delete(&model.LeaseCodeUsed{})
    DB.Where("used_at < ?", cutoff).Delete(&model.HostCodeUsed{})
}

// CleanOfflinePeers 30天没入网的设备删掉
func (as *ActivationService) CleanOfflinePeers() {
    cutoff := time.Now().Unix() - 30*86400
    var list []model.Peer
    DB.Where("last_online_time > 0 AND last_online_time < ?", cutoff).Find(&list)
    for _, p := range list {
        DB.Create(&model.RemovedDevice{
            Id:            p.Id,
            Account:       p.Account,
            LeaseExpireAt: p.LeaseExpireAt,
            HostExpireAt:  p.HostExpireAt,
            Ip:            p.LanIp,
            CreatedAt:     time.Time(p.CreatedAt).Unix(),
            RemovedAt:     time.Now().Unix(),
        })
        DB.Where("id = ?", p.Id).Delete(&model.AddressBook{})
        DB.Where("id = ?", p.Id).Delete(&model.Peer{})
    }
}