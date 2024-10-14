package apimodels

import "time"

type CatalogManifest struct {
	Name               string                       `json:"name"`
	ID                 string                       `json:"id"`
	CatalogID          string                       `json:"catalog_id"`
	Description        string                       `json:"description"`
	Architecture       string                       `json:"architecture"`
	Version            string                       `json:"version"`
	Type               string                       `json:"type"`
	Tags               []string                     `json:"tags"`
	Path               string                       `json:"path"`
	PackFilename       string                       `json:"pack_filename"`
	MetadataFilename   string                       `json:"metadata_filename"`
	Provider           CatalogManifestProvider      `json:"provider"`
	CreatedAt          time.Time                    `json:"created_at"`
	UpdatedAt          time.Time                    `json:"updated_at"`
	RequiredRoles      []string                     `json:"required_roles"`
	LastDownloadedAt   time.Time                    `json:"last_downloaded_at"`
	LastDownloadedUser string                       `json:"last_downloaded_user"`
	DownloadCount      int64                        `json:"download_count"`
	PackContents       []CatalogManifestPackContent `json:"pack_contents"`
}

type CatalogManifestPackContent struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type CatalogManifestProvider struct {
	Type string              `json:"type"`
	Meta CatalogManifestMeta `json:"meta"`
}

type CatalogManifestMeta struct {
	AccessKey string `json:"access_key"`
	Bucket    string `json:"bucket"`
	Region    string `json:"region"`
	SecretKey string `json:"secret_key"`
	User      string `json:"user"`
}
