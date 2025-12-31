package response

type UserResponseDTO struct {
	ID     uint   `json:"id"`
	Email  string `json:"email"`
	Prenom string `json:"prenom"`
	Nom    string `json:"nom"`
	Actif  bool   `json:"actif"`
}

type AuthResponseDTO struct {
	Token        string          `json:"token"`
	RefreshToken string          `json:"refreshToken"`
	User         UserResponseDTO `json:"user"`
}
