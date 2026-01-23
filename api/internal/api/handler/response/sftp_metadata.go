package response

type SftpMetadata struct {
	ID         uint   `json:"id"`
	Host       string `json:"host"`
	Port       string `json:"port"`
	User       string `json:"user"`
	Password   string `json:"password"`
	PrivateKey string `json:"privateKey"`
	BasePath   string `json:"basePath"`
	Extra      string `json:"extra"`
}
