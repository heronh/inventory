package models

import "gorm.io/gorm"

type Privilege struct {
	gorm.Model
	Description string `gorm:"size:50;uniqueIndex;not null"`
	Users       []User
}

type User struct {
	gorm.Model
	Name        string `gorm:"size:120;not null"`
	Email       string `gorm:"size:120;uniqueIndex;not null"`
	Password    string `gorm:"size:255;not null"`
	PrivilegeID uint
	Privilege   Privilege
	Phone       string `gorm:"size:30"`
	Tarefas     []Tarefas
}

type Tarefas struct {
	gorm.Model
	Description string `gorm:"type:text;not null"`
	Status      string `gorm:"size:30;not null"`
	UserID      uint
	User        User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type Client struct {
	gorm.Model
	Name         string `gorm:"size:120;not null"`
	Phone        string `gorm:"size:30"`
	TaxID        string `gorm:"size:30"`
	ZIPCode      string `gorm:"size:20"`
	Street       string `gorm:"size:120"`
	Number       string `gorm:"size:20"`
	Complement   string `gorm:"size:120"`
	Neighborhood string `gorm:"size:120"`
	City         string `gorm:"size:120"`
	State        string `gorm:"size:120"`
}

type Supplier struct {
	gorm.Model
	Name         string `gorm:"size:120;not null"`
	TradeName    string `gorm:"size:120"`
	TaxID        string `gorm:"size:30"`
	ZIPCode      string `gorm:"size:20"`
	Street       string `gorm:"size:120"`
	Number       string `gorm:"size:20"`
	Complement   string `gorm:"size:120"`
	Neighborhood string `gorm:"size:120"`
	City         string `gorm:"size:120"`
	State        string `gorm:"size:120"`
}

type Image struct {
	gorm.Model
	Name         string `gorm:"size:120;not null"`
	OriginalName string `gorm:"size:255;not null"`
	FullPath     string `gorm:"size:255;not null"`
}

type Product struct {
	gorm.Model
	Name         string  `gorm:"size:120;not null"`
	Description  string  `gorm:"type:text"`
	Images       []Image `gorm:"many2many:product_images;"`
	Code         string  `gorm:"size:60;uniqueIndex;not null"`
	Price        float64 `gorm:"type:numeric(14,2);not null"`
	Quantity     float64 `gorm:"type:numeric(14,2);not null"`
	Unit         string  `gorm:"size:20;not null"`
	MinimumStock float64 `gorm:"type:numeric(14,2);not null"`
}

type Entry struct {
	gorm.Model
	UserID      uint
	User        User
	Observation string `gorm:"type:text"`
	ProductID   uint
	Product     Product
	Quantity    float64 `gorm:"type:numeric(14,2);not null"`
	SupplierID  uint
	Supplier    Supplier
	Price       float64 `gorm:"type:numeric(14,2);not null"`
}

type Sale struct {
	gorm.Model
	UserID      uint
	User        User
	Observation string `gorm:"type:text"`
	ProductID   uint
	Product     Product
	Quantity    float64 `gorm:"type:numeric(14,2);not null"`
	ClientID    uint
	Client      Client
	Price       float64 `gorm:"type:numeric(14,2);not null"`
}

type Log struct {
	gorm.Model
	Description string `gorm:"type:text;not null"`
	UserID      uint
	User        User
}
