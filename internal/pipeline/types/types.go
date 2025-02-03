package types

import (
	"context"
	"time"
)

type BuildStatus string

const (
	BuildStatusPending   BuildStatus = "pending"
	BuildStatusBuilding  BuildStatus = "building"
	BuildStatusSuccess   BuildStatus = "success"
	BuildStatusFailed    BuildStatus = "failed"
	BuildStatusCancelled BuildStatus = "cancelled"
)

type Build struct {
	ID            string                 `json:"id"`
	ProjectID     string                 `json:"project_id"`
	CommitHash    string                 `json:"commit_hash"`
	Status        BuildStatus            `json:"status"`
	ImageID       string                 `json:"image_id,omitempty"`
	BuilderConfig map[string]interface{} `json:"builder_config"`
	Framework     string                 `json:"framework"`
	BuildCommand  string                 `json:"build_command"`
	OutputDir     string                 `json:"output_dir"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
	StartTime     time.Time              `json:"start_time"`
	CompleteTime  *time.Time             `json:"complete_time,omitempty"`
	ArtifactPath  string                 `json:"artifact_path,omitempty"`
	CancelFunc    context.CancelFunc     `json:"-"` // Internal use only`
}

type BuildResult struct {
	Success      bool
	ArtifactPath string
	ImageID      string
	Error        error
}
