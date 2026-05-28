package main

import "fmt"

// SystemInfoResponse represents the Emby /System/Info endpoint response
type SystemInfoResponse struct {
	SystemUpdateLevel string `json:"SystemUpdateLevel"`
	OperatingSystem   string `json:"OperatingSystem"`
	Version           string `json:"Version"`
	Id                string `json:"Id"`
	ServerName        string `json:"ServerName"`
}

// AuthByNameResponse represents the Emby /Users/AuthenticateByName endpoint response
type AuthByNameResponse struct {
	User        *AuthByNameUser `json:"User"`
	SessionInfo interface{}     `json:"SessionInfo"`
	AccessToken string          `json:"AccessToken"`
	ServerId    string          `json:"ServerId"`
}

// AuthByNameUser represents the user info in AuthenticateByName response
type AuthByNameUser struct {
	Name                  string `json:"Name"`
	Id                    string `json:"Id"`
	ServerId              string `json:"ServerId"`
	HasPassword           bool   `json:"HasPassword"`
	HasConfiguredPassword bool   `json:"HasConfiguredPassword"`
}

// ItemsResponse represents the Emby /Items endpoint response
type ItemsResponse struct {
	Items            []*EmbyItem `json:"Items"`
	TotalRecordCount int         `json:"TotalRecordCount"`
	StartIndex       int         `json:"StartIndex"`
}

// GenreItem represents a genre reference in an Emby response
type GenreItem struct {
	Name string      `json:"Name"`
	Id   interface{} `json:"Id"`
}

// EmbyItem represents a single item from the Emby API
type EmbyItem struct {
	Name           string            `json:"Name"`
	Id             string            `json:"Id"`
	Type           string            `json:"Type"`
	IsFolder       bool              `json:"IsFolder"`
	Path           string            `json:"Path"`
	ParentId       string            `json:"ParentId"`
	CollectionType string            `json:"CollectionType"`
	Overview       string            `json:"Overview"`
	RunTimeTicks   int64             `json:"RunTimeTicks"`
	ProductionYear int               `json:"ProductionYear"`
	IndexNumber    int               `json:"IndexNumber"`
	ParentIndexNum int               `json:"ParentIndexNumber"`
	SeriesName     string            `json:"SeriesName"`
	SeasonName     string            `json:"SeasonName"`
	Album          string            `json:"Album"`
	MediaType      string            `json:"MediaType"`
	Container      string            `json:"Container"`
	Size           int64             `json:"Size"`
	PremiereDate   string            `json:"PremiereDate"`
	DateCreated    string            `json:"DateCreated"`
	GenreItems     []*GenreItem      `json:"GenreItems"`
	ImageTags      map[string]string `json:"ImageTags"`
	UserData       *EmbyUserData     `json:"UserData"`
}

// EmbyUserData represents user-specific item data
type EmbyUserData struct {
	Played         bool   `json:"Played"`
	IsFavorite     bool   `json:"IsFavorite"`
	LastPlayedDate string `json:"LastPlayedDate"`
}

// UserResponse represents the /Users endpoint response (list of users)
type UserResponse struct {
	Name       string `json:"Name"`
	Id         string `json:"Id"`
	ServerId   string `json:"ServerId"`
	ServerName string `json:"ServerName"`
}

// GenresResponse represents the /Genres endpoint response
type GenresResponse struct {
	Items            []*EmbyItem `json:"Items"`
	TotalRecordCount int         `json:"TotalRecordCount"`
}

type ErrResponse struct {
	ErrorCode string `json:"ErrorCode"`
	Message   string `json:"Message"`
}

func (e ErrResponse) Error() string {
	return fmt.Sprintf("%s(%s)", e.Message, e.ErrorCode)
}
