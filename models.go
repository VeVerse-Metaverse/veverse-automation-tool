package main

import (
	"github.com/gofrs/uuid"
	"time"
)

// binarySuffixes list of suffixes of known|supported entrypoint binaries
var binarySuffixes = map[string]bool{
	"Server-Debug":     true,
	"Server-DebugGame": true,
	"Server":           true,
	"Server-Test":      true,
	"Server-Shipping":  true,
}

var SupportedJobTypes = map[string]bool{
	"Release":  true, // App release building and deployment
	"Package":  true, // Package processing
	"Launcher": true, // Launcher processing
}

var SupportedJobDeployments = map[string]bool{
	"Server": true,
	"Client": true,
	"SDK":    true,
}

type Identifier struct {
	Id *uuid.UUID `json:"id,omitempty"`
}

type EntityTrait struct {
	Identifier
	EntityId *uuid.UUID `json:"entityId,omitempty"`
}

type Timestamps struct {
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

type File struct {
	EntityTrait

	Type         string     `json:"type"`
	Url          string     `json:"url"`
	Mime         *string    `json:"mime,omitempty"`
	Size         *int64     `json:"size,omitempty"`
	Version      int        `json:"version,omitempty"`        // version of the file if versioned
	Deployment   string     `json:"deploymentType,omitempty"` // server or client if applicable
	Platform     string     `json:"platform,omitempty"`       // platform if applicable
	UploadedBy   *uuid.UUID `json:"uploadedBy,omitempty"`     // user that uploaded the file
	Width        *int       `json:"width,omitempty"`
	Height       *int       `json:"height,omitempty"`
	CreatedAt    time.Time  `json:"createdAt,omitempty"`
	UpdatedAt    *time.Time `json:"updatedAt,omitempty"`
	Variation    int        `json:"variation,omitempty"`    // variant of the file if applicable (e.g. PDF pages)
	OriginalPath string     `json:"originalPath,omitempty"` // original relative path to maintain directory structure (e.g. for releases)

	Timestamps
}

type Entity struct {
	Identifier
	EntityType *string `json:"entityType,omitempty"`
	Public     *bool   `json:"public,omitempty"`

	Timestamps

	Files []File `json:"files,omitempty"`
}

// Package entity model
type Package struct {
	Entity

	Name        string `json:"name,omitempty"`
	Title       string `json:"title,omitempty"`
	Summary     string `json:"summary,omitempty"`
	Description string `json:"description,omitempty"`
	Map         string `json:"map,omitempty"`
	Release     string `json:"release,omitempty"` // Based on
	Version     string `json:"version,omitempty"` // Version
}

// Release struct
type Release struct {
	Entity

	AppId          *uuid.UUID `json:"appId,omitempty"`
	AppName        string     `json:"appName,omitempty"`
	Version        string     `json:"version,omitempty"`
	CodeVersion    string     `json:"codeVersion,omitempty"`    // Code version
	ContentVersion string     `json:"contentVersion,omitempty"` // Content base version
	Map            string     `json:"map,omitempty"`
	Name           *string    `json:"name,omitempty"`
	Description    *string    `json:"description,omitempty"`
}

type App struct {
	Entity
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Url         string `json:"url,omitempty"`
	External    bool   `json:"external,omitempty"`
}

type JobMetadata struct {
	Entity
	EntityId      uuid.UUID `json:"entityId"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	OwnerId       uuid.UUID `json:"ownerId,omitempty"`
	WorkerId      uuid.UUID `json:"workerId,omitempty"`
	Configuration string    `json:"configuration,omitempty"`
	Platform      string    `json:"platform,omitempty"`
	Type          string    `json:"type,omitempty"`
	Deployment    string    `json:"deployment,omitempty"`
	Status        string    `json:"status,omitempty"`
	Message       string    `json:"message,omitempty"`
	App           *App      `json:"app,omitempty"`
	Package       *Package  `json:"package,omitempty"`
	Release       *Release  `json:"release,omitempty"`
}

type JobMetadataContainer struct {
	JobMetadata `json:"data"`
	Status      string `json:"status,omitempty"`
	Message     string `json:"message,omitempty"`
}

type JobStatusRequestMetadata struct {
	Status  string `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
}

type JobLogRequestMetadata struct {
	Warnings []string `json:"warnings,omitempty"`
	Errors   []string `json:"errors,omitempty"`
}
