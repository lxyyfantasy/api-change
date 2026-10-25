package model

// ===== 激活码 4 表 =====

type LeaseCodeUnused struct {
    Code     string `json:"code" gorm:"primaryKey;"`
    Days     int    `json:"days" gorm:"not null;"`
    ExpireAt int64  `json:"expire_at" gorm:"default:0;not null;"`
    TakenAt  int64  `json:"taken_at" gorm:"default:0;not null;"`
}
type LeaseCodeUsed struct {
    Code   string `json:"code" gorm:"primaryKey;"`
    Days   int    `json:"days" gorm:"not null;"`
    BindId string `json:"bind_id" gorm:"not null;index"`
    UsedAt int64  `json:"used_at" gorm:"not null;"`
}
type HostCodeUnused struct {
    Code     string `json:"code" gorm:"primaryKey;"`
    Days     int    `json:"days" gorm:"not null;"`
    ExpireAt int64  `json:"expire_at" gorm:"default:0;not null;"`
    TakenAt  int64  `json:"taken_at" gorm:"default:0;not null;"`
}
type HostCodeUsed struct {
    Code   string `json:"code" gorm:"primaryKey;"`
    Days   int    `json:"days" gorm:"not null;"`
    BindId string `json:"bind_id" gorm:"not null;index"`
    UsedAt int64  `json:"used_at" gorm:"not null;"`
}

// ===== 已删 ID 留存 =====

type RemovedDevice struct {
    Id            string `json:"id" gorm:"primaryKey;"`
    Account       string `json:"account" gorm:"not null;"`
    LeaseExpireAt int64  `json:"lease_expire_at" gorm:"default:0;not null;"`
    HostExpireAt  int64  `json:"host_expire_at" gorm:"default:0;not null;"`
    Ip            string `json:"ip" gorm:"default:'';not null;"`
    CreatedAt     int64  `json:"created_at" gorm:"default:0;not null;"`
    RemovedAt     int64  `json:"removed_at" gorm:"not null;"`
}

// ===== 已删账号留存 =====

type EmptyAccount struct {
    Account    string `json:"account" gorm:"primaryKey;"`
    EmptySince int64  `json:"empty_since" gorm:"not null;"`
}

// ===== 服务器配置（Key + ip1 + ip2） =====

type ServerConfig struct {
    Id         uint   `json:"id" gorm:"primaryKey;"`
    ServerKey  string `json:"server_key" gorm:"default:'';not null;"`
    Ip1        string `json:"ip1" gorm:"default:'';not null;"`
    Ip2        string `json:"ip2" gorm:"default:'';not null;"`
    UpdatedAt  int64  `json:"updated_at" gorm:"default:0;not null;"`
}