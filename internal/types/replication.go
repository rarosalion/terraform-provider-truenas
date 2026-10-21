package types

import (
	"encoding/json"
	"fmt"
)

// ReplicationSchedule is the schedule shape of a replication task. Unlike the
// plain Schedule it also carries the begin/end window, which TrueNAS always
// returns for replication tasks.
type ReplicationSchedule struct {
	Minute string `json:"minute"`
	Hour   string `json:"hour"`
	Dom    string `json:"dom"`
	Month  string `json:"month"`
	Dow    string `json:"dow"`
	Begin  string `json:"begin,omitempty"`
	End    string `json:"end,omitempty"`
}

// Replication represents a ZFS replication task.
type Replication struct {
	ID                      int                    `json:"id"`
	Name                    string                 `json:"name"`
	Direction               string                 `json:"direction"`
	Transport               string                 `json:"transport"`
	SourceDatasets          []string               `json:"source_datasets"`
	TargetDataset           string                 `json:"target_dataset"`
	Recursive               bool                   `json:"recursive"`
	AutoBool                bool                   `json:"auto"`
	Enabled                 bool                   `json:"enabled"`
	RetentionPolicy         string                 `json:"retention_policy"`
	LifetimeValue           int                    `json:"lifetime_value,omitempty"`
	LifetimeUnit            string                 `json:"lifetime_unit,omitempty"`
	Schedule                *ReplicationSchedule   `json:"schedule,omitempty"`
	SSHCredentials          int                    `json:"ssh_credentials,omitempty"`
	NamingSchema            []string               `json:"naming_schema,omitempty"`
	AlsoIncludeNamingSchema []string               `json:"also_include_naming_schema,omitempty"`
	Replicate               bool                   `json:"replicate"`
	Properties              bool                   `json:"properties"`
	PropertiesExclude       []string               `json:"properties_exclude,omitempty"`
	PropertiesOverride      map[string]interface{} `json:"properties_override,omitempty"`
	NameRegex               string                 `json:"name_regex,omitempty"`
	Exclude                 []string               `json:"exclude,omitempty"`
	Readonly                string                 `json:"readonly,omitempty"`
	AllowFromScratch        bool                   `json:"allow_from_scratch"`
	HoldPendingSnapshots    bool                   `json:"hold_pending_snapshots"`
	Retries                 int                    `json:"retries,omitempty"`
	LargeBlock              bool                   `json:"large_block"`
	Compressed              bool                   `json:"compressed"`
	Embed                   bool                   `json:"embed"`
	SpeedLimit              *int                   `json:"speed_limit,omitempty"`
	LoggingLevel            *string                `json:"logging_level,omitempty"`
	Sudo                    bool                   `json:"sudo"`
}

// UnmarshalJSON handles TrueNAS returning ssh_credentials as either a plain
// integer (create/update responses) or a nested keychain credential object
// {"id": N, ...} (get/query responses). The struct field always holds the
// integer ID. This is the same shape difference CloudSync.Credentials has.
func (r *Replication) UnmarshalJSON(data []byte) error {
	type Alias Replication
	aux := &struct {
		SSHCredentials json.RawMessage `json:"ssh_credentials"`
		*Alias
	}{
		Alias: (*Alias)(r),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	r.SSHCredentials = 0
	if len(aux.SSHCredentials) == 0 || string(aux.SSHCredentials) == "null" {
		return nil
	}
	var credID int
	if err := json.Unmarshal(aux.SSHCredentials, &credID); err == nil {
		r.SSHCredentials = credID
		return nil
	}
	var credObj struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(aux.SSHCredentials, &credObj); err != nil {
		return fmt.Errorf("ssh_credentials field: %w", err)
	}
	r.SSHCredentials = credObj.ID
	return nil
}

// ReplicationCreateRequest represents the request to create a replication task.
// The optional settings are pointers or omitempty so that a value left unset
// in the configuration is not sent and TrueNAS applies its own default.
type ReplicationCreateRequest struct {
	Name                    string                 `json:"name"`
	Direction               string                 `json:"direction"`
	Transport               string                 `json:"transport"`
	SourceDatasets          []string               `json:"source_datasets"`
	TargetDataset           string                 `json:"target_dataset"`
	Recursive               bool                   `json:"recursive"`
	AutoBool                bool                   `json:"auto"`
	Enabled                 bool                   `json:"enabled"`
	RetentionPolicy         string                 `json:"retention_policy"`
	LifetimeValue           int                    `json:"lifetime_value,omitempty"`
	LifetimeUnit            string                 `json:"lifetime_unit,omitempty"`
	Schedule                *ReplicationSchedule   `json:"schedule,omitempty"`
	SSHCredentials          int                    `json:"ssh_credentials,omitempty"`
	NamingSchema            []string               `json:"naming_schema,omitempty"`
	AlsoIncludeNamingSchema []string               `json:"also_include_naming_schema,omitempty"`
	Replicate               *bool                  `json:"replicate,omitempty"`
	Properties              *bool                  `json:"properties,omitempty"`
	PropertiesExclude       []string               `json:"properties_exclude,omitempty"`
	PropertiesOverride      map[string]interface{} `json:"properties_override,omitempty"`
	NameRegex               string                 `json:"name_regex,omitempty"`
	Exclude                 []string               `json:"exclude,omitempty"`
	Readonly                string                 `json:"readonly,omitempty"`
	AllowFromScratch        *bool                  `json:"allow_from_scratch,omitempty"`
	HoldPendingSnapshots    *bool                  `json:"hold_pending_snapshots,omitempty"`
	Retries                 *int                   `json:"retries,omitempty"`
	LargeBlock              *bool                  `json:"large_block,omitempty"`
	Compressed              *bool                  `json:"compressed,omitempty"`
	Embed                   *bool                  `json:"embed,omitempty"`
	SpeedLimit              *int                   `json:"speed_limit,omitempty"`
	LoggingLevel            string                 `json:"logging_level,omitempty"`
	Sudo                    *bool                  `json:"sudo,omitempty"`
}

// ReplicationUpdateRequest represents the request to update a replication task.
type ReplicationUpdateRequest struct {
	Name                 string                 `json:"name,omitempty"`
	Direction            string                 `json:"direction,omitempty"`
	Transport            string                 `json:"transport,omitempty"`
	SourceDatasets       []string               `json:"source_datasets,omitempty"`
	TargetDataset        string                 `json:"target_dataset,omitempty"`
	Recursive            *bool                  `json:"recursive,omitempty"`
	AutoBool             *bool                  `json:"auto,omitempty"`
	Enabled              *bool                  `json:"enabled,omitempty"`
	RetentionPolicy      string                 `json:"retention_policy,omitempty"`
	LifetimeValue        int                    `json:"lifetime_value,omitempty"`
	LifetimeUnit         string                 `json:"lifetime_unit,omitempty"`
	Schedule             *ReplicationSchedule   `json:"schedule,omitempty"`
	SSHCredentials       int                    `json:"ssh_credentials,omitempty"`
	Replicate            *bool                  `json:"replicate,omitempty"`
	Properties           *bool                  `json:"properties,omitempty"`
	PropertiesExclude    []string               `json:"properties_exclude,omitempty"`
	PropertiesOverride   map[string]interface{} `json:"properties_override,omitempty"`
	NameRegex            string                 `json:"name_regex,omitempty"`
	Exclude              []string               `json:"exclude,omitempty"`
	Readonly             string                 `json:"readonly,omitempty"`
	AllowFromScratch     *bool                  `json:"allow_from_scratch,omitempty"`
	HoldPendingSnapshots *bool                  `json:"hold_pending_snapshots,omitempty"`
	Retries              *int                   `json:"retries,omitempty"`
	LargeBlock           *bool                  `json:"large_block,omitempty"`
	Compressed           *bool                  `json:"compressed,omitempty"`
	Embed                *bool                  `json:"embed,omitempty"`
	SpeedLimit           *int                   `json:"speed_limit,omitempty"`
	LoggingLevel         string                 `json:"logging_level,omitempty"`
	Sudo                 *bool                  `json:"sudo,omitempty"`
}
