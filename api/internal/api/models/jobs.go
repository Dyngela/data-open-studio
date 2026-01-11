package models

type Job struct {
	ID          uint `gorm:"primaryKey"`
	Name        string
	Description string
	CreatorID   uint
	Active      bool
	Nodes       []Node `gorm:"foreignKey:JobID"`
	OutputPath  string
}
