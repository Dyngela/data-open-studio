package request

// DB Metadata DTOs

type CreateMetadata struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	User         string `json:"user"`
	Password     string `json:"password"`
	DatabaseName string `json:"databaseName"`
	SSLMode      string `json:"sslMode"`
}

type UpdateMetadata struct {
	Host         *string `json:"host,omitempty"`
	Port         *int    `json:"port,omitempty"`
	User         *string `json:"user,omitempty"`
	Password     *string `json:"password,omitempty"`
	DatabaseName *string `json:"databaseName,omitempty"`
	SSLMode      *string `json:"sslMode,omitempty"`
}

// SFTP Metadata DTOs

type CreateSftpMetadata struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	User       string `json:"user"`
	Password   string `json:"password"`
	PrivateKey string `json:"privateKey"`
	BasePath   string `json:"basePath"`
	Extra      string `json:"extra"`
}

type UpdateSftpMetadata struct {
	Host       *string `json:"host,omitempty"`
	Port       *int    `json:"port,omitempty"`
	User       *string `json:"user,omitempty"`
	Password   *string `json:"password,omitempty"`
	PrivateKey *string `json:"privateKey,omitempty"`
	BasePath   *string `json:"basePath,omitempty"`
	Extra      *string `json:"extra,omitempty"`
}

// DB Node DTOs

type GuessSchemaRequest struct {
	NodeID       string `json:"nodeId" validate:"required"`
	Query        string `json:"query" validate:"required"`
	ConnectionID uint   `json:"connectionId" validate:"required"`
}
