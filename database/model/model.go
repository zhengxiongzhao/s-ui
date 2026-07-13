package model

type Setting struct {
	Id    uint   `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	Key   string `json:"key" form:"key"`
	Value string `json:"value" form:"value"`
}

type Tls struct {
	Id     uint             `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	Name   string           `json:"name" form:"name"`
	Server JSONRawMessage   `json:"server" form:"server"`
	Client JSONRawMessage   `json:"client" form:"client"`
}

type User struct {
	Id         uint   `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	Username   string `json:"username" form:"username"`
	Password   string `json:"password" form:"password"`
	LastLogins string `json:"lastLogin"`
}

type Client struct {
	Id       uint           `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	Enable   bool           `json:"enable" form:"enable"`
	Name     string         `json:"name" form:"name"`
	Config   JSONRawMessage `json:"config,omitempty" form:"config"`
	Inbounds JSONRawMessage `json:"inbounds" form:"inbounds"`
	Nodes    JSONRawMessage `json:"nodes" form:"nodes"`
	Links    JSONRawMessage `json:"links,omitempty" form:"links"`
	Volume   int64           `json:"volume" form:"volume"`
	Expiry   int64           `json:"expiry" form:"expiry"`
	Down     int64           `json:"down" form:"down"`
	Up       int64           `json:"up" form:"up"`
	Desc     string          `json:"desc" form:"desc"`
	Group    string          `json:"group" form:"group"`
	Remark   string          `json:"remark" form:"remark"`

	// Timestamps (unix seconds): creation time and last time the client had traffic
	CreatedAt int64 `json:"createdAt" form:"createdAt" gorm:"default:0;not null"`
	OnlineAt  int64 `json:"onlineAt" form:"onlineAt" gorm:"default:0;not null"`

	// Delay start and periodic reset
	DelayStart bool  `json:"delayStart" form:"delayStart" gorm:"default:false;not null"`
	AutoReset  bool  `json:"autoReset" form:"autoReset" gorm:"default:false;not null"`
	ResetDays  int   `json:"resetDays" form:"resetDays" gorm:"default:0;not null"`
	NextReset  int64 `json:"nextReset" form:"nextReset" gorm:"default:0;not null"`
	TotalUp    int64 `json:"totalUp" form:"totalUp" gorm:"default:0;not null"`
	TotalDown  int64 `json:"totalDown" form:"totalDown" gorm:"default:0;not null"`
}

type Stats struct {
	Id        uint64 `json:"id" gorm:"primaryKey;autoIncrement"`
	DateTime  int64  `json:"dateTime" gorm:"uniqueIndex:idx_stats_bucket,priority:3"`
	Resource  string `json:"resource" gorm:"uniqueIndex:idx_stats_bucket,priority:1"`
	Tag       string `json:"tag" gorm:"uniqueIndex:idx_stats_bucket,priority:2"`
	Direction bool   `json:"direction" gorm:"uniqueIndex:idx_stats_bucket,priority:4"`
	Traffic   int64  `json:"traffic"`
	NodeId    uint   `json:"nodeId" gorm:"default:1;not null"`
}

type Changes struct {
	Id       uint64         `json:"id" gorm:"primaryKey;autoIncrement"`
	DateTime int64          `json:"dateTime"`
	Actor    string         `json:"actor"`
	Key      string         `json:"key"`
	Action   string         `json:"action"`
	Obj      JSONRawMessage `json:"obj"`
}

type Tokens struct {
	Id     uint   `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	Desc   string `json:"desc" form:"desc"`
	Token  string `json:"token" form:"token"`
	Expiry int64  `json:"expiry" form:"expiry"`
	UserId uint   `json:"userId" form:"userId"`
	User   *User  `json:"user" gorm:"foreignKey:UserId;references:Id"`
}
