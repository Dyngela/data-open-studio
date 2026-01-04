package request

type CreateMetadata struct {
	Host         string `json:"host"`
	Port         string `json:"port"`
	User         string `json:"user"`
	Password     string `json:"password"`
	DatabaseName string `json:"databaseName"`
	SSLMode      string `json:"sslMode"`
}

type UpdateMetadata struct {
	ID           uint    `json:"id;required"`
	Host         *string `json:"host,omitempty"`
	Port         *string `json:"port,omitempty"`
	User         *string `json:"user,omitempty"`
	Password     *string `json:"password,omitempty"`
	DatabaseName *string `json:"databaseName,omitempty"`
	SSLMode      *string `json:"sslMode,omitempty"`
}
