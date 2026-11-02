package models

import (
	"time"

	"gorm.io/gorm"
)

// Global roles
const (
	RoleAdmin    = "admin"
	RoleOperator = "operator"
	RoleMember   = "member"
)

// Membership roles within a vscale
const (
	MembershipOwner  = "owner"
	MembershipAdmin  = "admin"
	MembershipMember = "member"
)

const (
	StatusActive   = "active"
	StatusDisabled = "disabled"
	StatusRevoked  = "revoked"
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusDenied   = "denied"
	StatusOnline   = "online"
	StatusOffline  = "offline"
)

const (
	RouteViaHub    = "hub"
	RouteViaDevice = "device"
)

const (
	AllocKindOverlay = "overlay" // vscale CIDR block
	AllocKindDevice  = "device"  // device /32
	AllocKindRoute   = "route"   // advertised subnet
	AllocKindHub     = "hub"     // hub address inside overlay
)

type User struct {
	gorm.Model
	Username string `gorm:"uniqueIndex;not null" json:"username"`
	Password string `gorm:"not null" json:"-"`
	Email    string `gorm:"uniqueIndex;not null" json:"email"`
	Role     string `gorm:"not null;default:'member'" json:"role"`
}

type Vscale struct {
	gorm.Model
	Name        string `gorm:"uniqueIndex;not null" json:"name"`
	Description string `gorm:"type:text" json:"description"`
	OverlayCIDR string `gorm:"not null" json:"overlayCidr"` // e.g. 100.64.0.0/16
	DNS         string `json:"dns"`
	Status      string `gorm:"not null;default:'active'" json:"status"`
	Hubs        []Hub        `json:"hubs,omitempty"`
	Devices     []Device     `json:"devices,omitempty"`
	Memberships []Membership `json:"memberships,omitempty"`
	Routes      []Route      `json:"routes,omitempty"`
}

type Hub struct {
	gorm.Model
	VscaleID      uint       `gorm:"not null;index" json:"vscaleId"`
	Vscale        Vscale     `json:"-"`
	Name          string     `gorm:"not null" json:"name"`
	PublicKey     string     `gorm:"not null" json:"publicKey"`
	PrivateKey    string     `gorm:"not null" json:"-"`
	ListenPort    int        `gorm:"not null;default:51820" json:"listenPort"`
	Endpoint      string     `json:"endpoint"` // host:port or host
	Address       string     `gorm:"not null" json:"address"` // overlay IP /32
	MTU           int        `gorm:"default:1420" json:"mtu"`
	Status        string     `gorm:"not null;default:'active'" json:"status"`
	AgentToken    string     `gorm:"uniqueIndex" json:"-"`
	LastHeartbeat *time.Time `json:"lastHeartbeat,omitempty"`
	TransferRx    int64      `json:"transferRx"`
	TransferTx    int64      `json:"transferTx"`
}

type Device struct {
	gorm.Model
	VscaleID      uint       `gorm:"not null;index" json:"vscaleId"`
	Vscale        Vscale     `json:"-"`
	OwnerUserID   *uint      `json:"ownerUserId,omitempty"`
	Owner         *User      `gorm:"foreignKey:OwnerUserID" json:"owner,omitempty"`
	Name          string     `gorm:"not null" json:"name"`
	Hostname      string     `json:"hostname"`
	OS            string     `json:"os"`
	PublicKey     string     `gorm:"not null;uniqueIndex" json:"publicKey"`
	PrivateKey    string     `gorm:"not null" json:"-"`
	Address       string     `gorm:"not null" json:"address"` // overlay /32
	DNS           string     `json:"dns"`
	Tags          string     `gorm:"type:text" json:"tags"`
	Status        string     `gorm:"not null;default:'active'" json:"status"`
	AgentToken    string     `gorm:"uniqueIndex" json:"-"`
	LastHeartbeat *time.Time `json:"lastHeartbeat,omitempty"`
	LastHandshake *time.Time `json:"lastHandshake,omitempty"`
	TransferRx    int64      `json:"transferRx"`
	TransferTx    int64      `json:"transferTx"`
	LocalCIDRs    string     `gorm:"type:text" json:"localCidrs,omitempty"` // reported by agent
}

type Membership struct {
	gorm.Model
	VscaleID uint   `gorm:"not null;uniqueIndex:idx_member_vscale_user" json:"vscaleId"`
	UserID   uint   `gorm:"not null;uniqueIndex:idx_member_vscale_user" json:"userId"`
	Role     string `gorm:"not null;default:'member'" json:"role"`
	User     User   `json:"user,omitempty"`
	Vscale   Vscale `json:"-"`
}

type JoinToken struct {
	gorm.Model
	VscaleID   uint       `gorm:"not null;index" json:"vscaleId"`
	Vscale     Vscale     `json:"-"`
	CreatedBy  uint       `json:"createdBy"`
	Token      string     `gorm:"uniqueIndex;not null" json:"token"`
	Label      string     `json:"label"`
	Reusable   bool       `gorm:"default:false" json:"reusable"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
	UsesLeft   int        `gorm:"default:1" json:"usesLeft"`
	Revoked    bool       `gorm:"default:false" json:"revoked"`
}

type Route struct {
	gorm.Model
	VscaleID    uint   `gorm:"not null;index" json:"vscaleId"`
	Vscale      Vscale `json:"-"`
	CIDR        string `gorm:"not null" json:"cidr"`
	ViaType     string `gorm:"not null" json:"viaType"` // hub | device
	ViaID       uint   `gorm:"not null" json:"viaId"`
	Status      string `gorm:"not null;default:'pending'" json:"status"`
	Description string `json:"description"`
	Advertised  bool   `gorm:"default:true" json:"advertised"`
}

type IpAllocation struct {
	gorm.Model
	VscaleID  *uint  `gorm:"index" json:"vscaleId,omitempty"`
	Kind      string `gorm:"not null;index" json:"kind"`
	CIDR      string `gorm:"not null;index" json:"cidr"`
	RefType   string `json:"refType"` // vscale|hub|device|route
	RefID     uint   `json:"refId"`
	Note      string `json:"note"`
}

// MetricSnapshot is a periodic rollup for dashboard time-series charts.
type MetricSnapshot struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	TakenAt        time.Time `gorm:"index;not null" json:"takenAt"`
	Vscales        int64     `json:"vscales"`
	Hubs           int64     `json:"hubs"`
	HubsOnline     int64     `json:"hubsOnline"`
	Devices        int64     `json:"devices"`
	DevicesOnline  int64     `json:"devicesOnline"`
	Routes         int64     `json:"routes"`
	TransferRx     int64     `json:"transferRx"`
	TransferTx     int64     `json:"transferTx"`
	TransferRxRate int64     `json:"transferRxRate"` // bytes/sec since previous sample
	TransferTxRate int64     `json:"transferTxRate"`
}
