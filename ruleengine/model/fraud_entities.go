package model

import "fmt"

// 反欺诈场景的业务实体定义

// User 用户实体
type User struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Status  string `json:"status"` // "normal", "locked", "suspicious"
	Level   string `json:"level"`  // "VIP", "normal"
	Country string `json:"country"`
}

func (u User) Key() string { return fmt.Sprintf("User:%d", u.ID) }

// Account 账户实体
type Account struct {
	ID       int     `json:"id"`
	UserID   int     `json:"user_id"`
	Balance  float64 `json:"balance"`
	Currency string  `json:"currency"`
	Status   string  `json:"status"` // "active", "frozen"
}

func (a Account) Key() string { return fmt.Sprintf("Account:%d", a.ID) }

// Transaction 交易实体
type Transaction struct {
	ID       int     `json:"id"`
	UserID   int     `json:"user_id"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Type     string  `json:"type"`     // "deposit", "withdraw", "transfer"
	Status   string  `json:"status"`   // "pending", "completed", "failed"
	Location string  `json:"location"` // 交易地点
}

func (t Transaction) Key() string { return fmt.Sprintf("Transaction:%d", t.ID) }

// LoginAttempt 登录尝试实体
type LoginAttempt struct {
	ID        int    `json:"id"`
	UserID    int    `json:"user_id"`
	Success   bool   `json:"success"`
	Timestamp int64  `json:"timestamp"`
	IP        string `json:"ip"`
	Location  string `json:"location"`
}

func (l LoginAttempt) Key() string { return fmt.Sprintf("LoginAttempt:%d", l.ID) }

// SecurityAlert 安全警报实体（规则触发后生成）
type SecurityAlert struct {
	ID      int    `json:"id"`
	UserID  int    `json:"user_id"`
	Type    string `json:"type"` // "fraud_risk", "account_locked", "suspicious_login"
	Message string `json:"message"`
	Level   string `json:"level"` // "low", "medium", "high", "critical"
}

func (s SecurityAlert) Key() string { return fmt.Sprintf("SecurityAlert:%d", s.ID) }

// Cart 购物车实体（用于优惠券场景）
type Cart struct {
	ID         int     `json:"id"`
	UserID     int     `json:"user_id"`
	TotalValue float64 `json:"total_value"`
}

func (c Cart) Key() string { return fmt.Sprintf("Cart:%d", c.ID) }

// UserProfile 用户画像实体
type UserProfile struct {
	UserID          int    `json:"user_id"`
	RegistrationAge int    `json:"registration_age"` // 注册天数
	ActivityLevel   string `json:"activity_level"`   // "low", "normal", "high"
	RiskScore       int    `json:"risk_score"`       // 0-100风险评分
	HomeLocation    string `json:"home_location"`    // 常用地址
}

func (up UserProfile) Key() string { return fmt.Sprintf("UserProfile:%d", up.UserID) }

// FailedAttempt 失败尝试实体（用于聚合统计）
type FailedAttempt struct {
	ID     int    `json:"id"`
	UserID int    `json:"user_id"`
	Type   string `json:"type"` // "login", "payment", "withdrawal"
	Count  int    `json:"count"`
}

func (fa FailedAttempt) Key() string { return fmt.Sprintf("FailedAttempt:%d", fa.ID) }

// DeviceInfo 设备信息实体
type DeviceInfo struct {
	DeviceID   string `json:"device_id"`
	UserID     int    `json:"user_id"`
	Trusted    bool   `json:"trusted"`
	LastSeen   int64  `json:"last_seen"`
	DeviceType string `json:"device_type"` // "mobile", "desktop", "tablet"
}

func (di DeviceInfo) Key() string { return fmt.Sprintf("DeviceInfo:%s", di.DeviceID) }
