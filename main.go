package main

import (
	ops "Users/alexeylychkin/Desktop/NeoToolsBackend/draft/models"
	service "Users/alexeylychkin/Desktop/NeoToolsBackend/draft/services"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// migrateDB()
	// ops.InitUnionOne()
	// service.ComputeAutoOperationsFromDB()
	// lib.PrintResp(service.GetFundTreeByManualOperationID(3))
	service.GetFundTreeByManualOperationID(3, false)
	service.GetManualOperations(true)
	//TODO:
	// 1.Сервис работы с операциями: добавить мануальную операцию
	// 2.Список статичных расходов внутри фондов CRUD
	// 3.Список сотрудников и их компетенций (Юнион/Команда/Функция/Компетенции/Уровень, база, человек, балл)

}

func migrateDB() {
	db, err := gorm.Open(sqlite.Open("gorm.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	//https://gorm.io/docs/migration.html
	db.Migrator().DropTable(
		&ops.Sprint{},
		&ops.Team{},
		&ops.Union{},
		&ops.Goals{},
		&ops.IncomeAccount{},
		&ops.Partner{},
		&ops.OperationStatus{},
		&ops.Fund{},
		&ops.ManualOperation{},
		&ops.AutoOperation{},
	)
	db.AutoMigrate(
		&ops.Sprint{},
		&ops.Team{},
		&ops.Union{},
		&ops.Goals{},
		&ops.IncomeAccount{},
		&ops.Partner{},
		&ops.OperationStatus{},
		&ops.Fund{},
		&ops.ManualOperation{},
		&ops.AutoOperation{},
	)
}
