package request

type RegisterDTO struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Prenom   string `json:"prenom" validate:"required"`
	Nom      string `json:"nom" validate:"required"`
}

type LoginDTO struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RefreshTokenDTO struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
}

type UpdateUser struct {
	Email  *string `json:"email"`
	Prenom *string `json:"prenom"`
	Nom    *string `json:"nom"`
	Actif  *bool   `json:"actif"`
}
