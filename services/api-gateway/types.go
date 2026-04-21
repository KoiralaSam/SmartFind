package main

import "time"

type PassengerLoginRequest struct {
	IDToken string `json:"id_token"`
}

type PassengerDTO struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FullName  string `json:"full_name"`
	Phone     string `json:"phone"`
	AvatarURL string `json:"avatar_url"`
}

type PassengerLoginResponse struct {
	Passenger    PassengerDTO `json:"passenger"`
	SessionToken string       `json:"session_token"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type DetailExtractRequest struct {
	ImageBase64 string `json:"image_base64"`
}

type DetailExtractResponse struct {
	ItemName        string `json:"item_name"`
	ItemType        string `json:"item_type"`
	Category        string `json:"category"`
	Brand           string `json:"brand"`
	Model           string `json:"model"`
	Color           string `json:"color"`
	Material        string `json:"material"`
	ItemCondition   string `json:"item_condition"`
	ItemDescription string `json:"item_description"`
}

type StaffDTO struct {
	ID        string    `json:"id"`
	FullName  string    `json:"full_name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type StaffLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type StaffLoginResponse struct {
	Staff        StaffDTO `json:"staff"`
	SessionToken string   `json:"session_token"`
}

type StaffCreateStaffRequest struct {
	TransitCode string `json:"transit_code"`
	FullName    string `json:"full_name"`
	Email       string `json:"email"`
	Password    string `json:"password"`
}

type StaffCreateFoundItemRequest struct {
	StaffID         string   `json:"staff_id"`
	ItemName        string   `json:"item_name"`
	ItemDescription string   `json:"item_description"`
	ItemType        string   `json:"item_type"`
	Brand           string   `json:"brand"`
	Model           string   `json:"model"`
	Color           string   `json:"color"`
	Material        string   `json:"material"`
	ItemCondition   string   `json:"item_condition"`
	Category        string   `json:"category"`
	LocationFound   string   `json:"location_found"`
	RouteOrStation  string   `json:"route_or_station"`
	RouteID         string   `json:"route_id"`
	DateFound       string   `json:"date_found"` // RFC3339
	ImageKeys       []string `json:"image_keys"`
	PrimaryImageKey string   `json:"primary_image_key"`
}

type StaffUpdateFoundItemStatusRequest struct {
	StaffID     string `json:"staff_id"`
	FoundItemID string `json:"found_item_id"`
	Status      string `json:"status"`
}

// StaffUpdateFoundItemRequest carries the patch body for PUT /staff/found-items/{id}.
// Only non-empty fields are applied; image_keys replaces the full list when present.
type StaffUpdateFoundItemRequest struct {
	StaffID         string   `json:"staff_id"`
	ItemName        string   `json:"item_name"`
	ItemDescription string   `json:"item_description"`
	ItemType        string   `json:"item_type"`
	Brand           string   `json:"brand"`
	Model           string   `json:"model"`
	Color           string   `json:"color"`
	Material        string   `json:"material"`
	ItemCondition   string   `json:"item_condition"`
	Category        string   `json:"category"`
	LocationFound   string   `json:"location_found"`
	RouteOrStation  string   `json:"route_or_station"`
	RouteID         string   `json:"route_id"`
	DateFound       string   `json:"date_found"` // RFC3339; omit to leave unchanged
	ImageKeys       []string `json:"image_keys"`
	PrimaryImageKey string   `json:"primary_image_key"`
}

// StaffDeleteFoundItemRequest carries the body for DELETE /staff/found-items/{id}.
type StaffDeleteFoundItemRequest struct {
	StaffID string `json:"staff_id"`
}

type FoundItemDTO struct {
	ID              string    `json:"id"`
	PostedByStaffID string    `json:"posted_by_staff_id"`
	ItemName        string    `json:"item_name"`
	ItemDescription string    `json:"item_description"`
	ItemType        string    `json:"item_type"`
	Brand           string    `json:"brand"`
	Model           string    `json:"model"`
	Color           string    `json:"color"`
	Material        string    `json:"material"`
	ItemCondition   string    `json:"item_condition"`
	Category        string    `json:"category"`
	LocationFound   string    `json:"location_found"`
	RouteOrStation  string    `json:"route_or_station"`
	RouteID         string    `json:"route_id"`
	DateFound       time.Time `json:"date_found"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	Image           string    `json:"image"`
	Images          []string  `json:"images"`
}

type StaffListFoundItemsResponse struct {
	Items []FoundItemDTO `json:"items"`
}

// PassengerClaimFoundItemDTO mirrors FoundItemMatch JSON for the passenger UI (images + details).
type PassengerClaimFoundItemDTO struct {
	FoundItemID     string    `json:"found_item_id"`
	ItemName        string    `json:"item_name"`
	ItemDescription string    `json:"item_description"`
	ItemType        string    `json:"item_type"`
	Brand           string    `json:"brand"`
	Model           string    `json:"model"`
	Color           string    `json:"color"`
	Material        string    `json:"material"`
	ItemCondition   string    `json:"item_condition"`
	Category        string    `json:"category"`
	LocationFound   string    `json:"location_found"`
	RouteOrStation  string    `json:"route_or_station"`
	RouteID         string    `json:"route_id"`
	DateFound       time.Time `json:"date_found"`
	Status          string    `json:"status"`
	SimilarityScore float64   `json:"similarity_score"`
	ImageURLs       []string  `json:"image_urls"`
	PrimaryImageURL string    `json:"primary_image_url"`
}

type ItemClaimDTO struct {
	ID                  string                      `json:"id"`
	ItemID              string                      `json:"item_id"`
	ClaimantPassengerID string                      `json:"claimant_passenger_id"`
	ClaimantName        string                      `json:"claimant_name"`
	ClaimantEmail       string                      `json:"claimant_email"`
	LostReportID        string                      `json:"lost_report_id"`
	Message             string                      `json:"message"`
	Status              string                      `json:"status"`
	CreatedAt           time.Time                   `json:"created_at"`
	UpdatedAt           time.Time                   `json:"updated_at"`
	FoundItem           *PassengerClaimFoundItemDTO `json:"found_item,omitempty"`
}

type StaffListClaimsResponse struct {
	Claims []ItemClaimDTO `json:"claims"`
}

type StaffReviewClaimRequest struct {
	StaffID  string `json:"staff_id"`
	ClaimID  string `json:"claim_id"`
	Decision string `json:"decision"`
}

type RouteDTO struct {
	ID               string    `json:"id"`
	RouteName        string    `json:"route_name"`
	CreatedByStaffID string    `json:"created_by_staff_id"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type StaffCreateRouteRequest struct {
	StaffID   string `json:"staff_id"`
	RouteName string `json:"route_name"`
}

type StaffDeleteRouteRequest struct {
	StaffID string `json:"staff_id"`
	RouteID string `json:"route_id"`
}

type StaffListRoutesResponse struct {
	Routes []RouteDTO `json:"routes"`
}
