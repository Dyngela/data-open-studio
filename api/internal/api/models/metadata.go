package models

type MetadataDatabase struct {
	ID           uint   `json:"id"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	User         string `json:"user"`
	Password     string `json:"password"`
	DatabaseName string `json:"databaseName"`
	SSLMode      string `json:"sslMode"`
	Extra        string `json:"extra"`
	DbType       DBType `json:"dbName" gorm:"column:db_type;type:varchar(20)"`
}

type MetadataSftp struct {
	ID         uint   `json:"id"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	User       string `json:"user"`
	Password   string `json:"password"`
	PrivateKey string `json:"privateKey"`
	BasePath   string `json:"basePath"`
	Extra      string `json:"extra"`
}
