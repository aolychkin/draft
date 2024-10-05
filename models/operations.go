package models

import (
	"time"

	"gorm.io/gorm"
)

type ManualOperation struct {
	gorm.Model
	Date              time.Time `gorm:"not null"`
	Value             float32   `gorm:"not null"`
	Details           string
	TeamID            string
	FundID            string
	IncomeAccountID   string
	PartnerID         string
	OperationStatusID string
	AutoOperation     []AutoOperation
}

type AutoOperation struct {
	gorm.Model
	Date              time.Time `gorm:"not null"`
	Value             float32   `gorm:"not null"`
	ManualOperationID string
	FundID            string
	GoalsID           string
	OperationStatusID string
}

type OperationStatus struct {
	gorm.Model
	Label           string `gorm:"not null"`
	ManualOperation []ManualOperation
}

type Partner struct {
	gorm.Model
	Label           string `gorm:"not null"`
	ManualOperation []ManualOperation
}

type IncomeAccount struct {
	gorm.Model
	Label           string `gorm:"not null"`
	Bank            string
	UnionID         string
	ManualOperation []ManualOperation
}

type Sprint struct {
	gorm.Model
	Number    uint      `gorm:"not null"`
	StartDate time.Time `gorm:"not null"`
	EndDate   time.Time `gorm:"not null"`
	TeamID    string
}

type Team struct {
	gorm.Model
	Label           string `gorm:"not null"`
	UnionID         string `gorm:"not null"`
	Sprint          []Sprint
	ManualOperation []ManualOperation
}

type Union struct {
	gorm.Model
	Label         string `gorm:"not null"`
	IncomeAccount []IncomeAccount
	Team          []Team
}

// RuleValue = процент, который от родительского фонда / операции (!= null) распределяется по детям
// Если RuleValue = -1 и CheckChild = true, вычисляется от детей. То есть сумма их значений
// Если в детях есть правило %, то смотрится родитель фонда, у которого был CheckChild = true
// ___________________
// Если RuleValue = -1 и CheckChild = false и нет Goals, вычисляется по остаточному принципу
// Если RuleValue = -1 и CheckChild = false и есть Goals, вычисляется по целям
// Если RuleValue > 0 и CheckChild = false, то от заданного RuleValue, рассчитывается значение детей
// ___________________
// Если RuleValue = 0. то в фонд НЕ идет отчислений, проверку детей - пропускаем
type Fund struct {
	gorm.Model
	Label           string  `gorm:"not null"`
	Priority        uint    `gorm:"not null"`
	Union           []Union `gorm:"many2many:fund_union"`
	CheckChild      bool
	RuleValue       float32
	Goals           []Goals
	Child           []*Fund `gorm:"many2many:fund_child"`
	ManualOperation []ManualOperation
	AutoOperation   []AutoOperation
}

// ExpireDate = срок возврата инвестиций
type Goals struct {
	gorm.Model
	Label         string
	Total         float32
	ExpireDate    time.Time
	FundID        string
	AutoOperation []AutoOperation
}
