package models

import (
	"time"

	"github.com/google/uuid"
)

// Role represents an RBAC role within an organization.
type Role string

const (
	RoleAdmin    Role = "admin"    // full control within the tenant
	RoleOperator Role = "operator" // runs engagements/campaigns
	RoleViewer   Role = "viewer"   // read-only (e.g. client stakeholder)
)

func (r Role) Valid() bool {
	switch r {
	case RoleAdmin, RoleOperator, RoleViewer:
		return true
	}
	return false
}

// AtLeast reports whether r is allowed where min is required (admin>operator>viewer).
func (r Role) AtLeast(min Role) bool {
	rank := map[Role]int{RoleViewer: 1, RoleOperator: 2, RoleAdmin: 3}
	return rank[r] >= rank[min]
}

type Organization struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type User struct {
	ID           uuid.UUID `json:"id"`
	OrgID        uuid.UUID `json:"org_id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         Role      `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

// EngagementStatus lifecycle.
type EngagementStatus string

const (
	EngagementDraft    EngagementStatus = "draft"
	EngagementActive   EngagementStatus = "active"
	EngagementClosed   EngagementStatus = "closed"
)

// Engagement is the authorization record. Campaigns can only run inside an
// active engagement whose date window contains "now".
type Engagement struct {
	ID         uuid.UUID        `json:"id"`
	OrgID      uuid.UUID        `json:"org_id"`
	ClientName string           `json:"client_name"`
	AuthzRef   string           `json:"authz_ref"` // reference to signed authorization
	StartsAt   time.Time        `json:"starts_at"`
	EndsAt     time.Time        `json:"ends_at"`
	Status     EngagementStatus `json:"status"`
	CreatedAt  time.Time        `json:"created_at"`
}

// Active reports whether the engagement may currently send.
func (e Engagement) Active(now time.Time) bool {
	return e.Status == EngagementActive && !now.Before(e.StartsAt) && !now.After(e.EndsAt)
}

// ScopeRule is an allowlist entry. Only targets matching a rule may be contacted.
type ScopeRule struct {
	ID           uuid.UUID `json:"id"`
	EngagementID uuid.UUID `json:"engagement_id"`
	Kind         string    `json:"kind"`    // "domain" | "email"
	Pattern      string    `json:"pattern"` // domain suffix, or glob email
	CreatedAt    time.Time `json:"created_at"`
}

type SendingProfile struct {
	ID           uuid.UUID `json:"id"`
	OrgID        uuid.UUID `json:"org_id"`
	Name         string    `json:"name"`
	SMTPHost     string    `json:"smtp_host"`
	SMTPPort     int       `json:"smtp_port"`
	Username     string    `json:"username"`
	Password     string    `json:"-"` // never serialized
	FromAddress  string    `json:"from_address"`
	FromName     string    `json:"from_name"`
	UseTLS       bool      `json:"use_tls"`
	CreatedAt    time.Time `json:"created_at"`
}

type Target struct {
	ID           uuid.UUID `json:"id"`
	EngagementID uuid.UUID `json:"engagement_id"`
	Email        string    `json:"email"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Position     string    `json:"position"`
	Timezone     string    `json:"timezone"`
	CreatedAt    time.Time `json:"created_at"`
}

type EmailTemplate struct {
	ID        uuid.UUID `json:"id"`
	OrgID     uuid.UUID `json:"org_id"`
	Name      string    `json:"name"`
	Subject   string    `json:"subject"`
	HTML      string    `json:"html"`
	Text      string    `json:"text"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
}

type LandingPage struct {
	ID          uuid.UUID `json:"id"`
	OrgID       uuid.UUID `json:"org_id"`
	Name        string    `json:"name"`
	HTML        string    `json:"html"`
	CaptureMeta bool      `json:"capture_meta"` // capture which fields were filled (never values)
	RedirectURL string    `json:"redirect_url"` // awareness training page
	CreatedAt   time.Time `json:"created_at"`
}

type CampaignStatus string

const (
	CampaignDraft     CampaignStatus = "draft"
	CampaignScheduled CampaignStatus = "scheduled"
	CampaignRunning   CampaignStatus = "running"
	CampaignDone      CampaignStatus = "completed"
)

type Campaign struct {
	ID              uuid.UUID      `json:"id"`
	EngagementID    uuid.UUID      `json:"engagement_id"`
	Name            string         `json:"name"`
	EmailTemplateID uuid.UUID      `json:"email_template_id"`
	LandingPageID   uuid.UUID      `json:"landing_page_id"`
	SendingProfileID uuid.UUID     `json:"sending_profile_id"`
	Status          CampaignStatus `json:"status"`
	LaunchAt        *time.Time     `json:"launch_at"`
	RatePerMinute   int            `json:"rate_per_minute"`
	CreatedAt       time.Time      `json:"created_at"`
}

// CampaignTarget links a campaign to a target and carries the signed tracking id.
type CampaignTarget struct {
	ID         uuid.UUID `json:"id"`
	CampaignID uuid.UUID `json:"campaign_id"`
	TargetID   uuid.UUID `json:"target_id"`
	RID        string    `json:"rid"`    // opaque HMAC-signed id used in tracking links
	Status     string    `json:"status"` // pending|sent|error
	Error      string    `json:"error,omitempty"`
}

// EventType captures the phishing-simulation funnel.
type EventType string

const (
	EventSent   EventType = "sent"
	EventOpen   EventType = "open"
	EventClick  EventType = "click"
	EventSubmit EventType = "submit"
	EventReport EventType = "report"
)

type Event struct {
	ID               uuid.UUID `json:"id"`
	CampaignTargetID uuid.UUID `json:"campaign_target_id"`
	Type             EventType `json:"type"`
	IP               string    `json:"ip"`
	UserAgent        string    `json:"user_agent"`
	Meta             string    `json:"meta"` // JSON; for submit: field names only, never values
	CreatedAt        time.Time `json:"created_at"`
}

type AuditEntry struct {
	ID        uuid.UUID `json:"id"`
	OrgID     uuid.UUID `json:"org_id"`
	ActorID   *uuid.UUID `json:"actor_id"`
	Action    string    `json:"action"`
	Entity    string    `json:"entity"`
	EntityID  string    `json:"entity_id"`
	Meta      string    `json:"meta"`
	CreatedAt time.Time `json:"created_at"`
}

type Webhook struct {
	ID        uuid.UUID `json:"id"`
	OrgID     uuid.UUID `json:"org_id"`
	URL       string    `json:"url"`
	Secret    string    `json:"-"`
	Events    []string  `json:"events"`
	CreatedAt time.Time `json:"created_at"`
}
