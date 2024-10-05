
// type tmpSprints struct {
// 	Number    uint
// 	StartDate time.Time
// 	EndDate   time.Time
// }
// func fundSprint() []tmpSprints {
// 	var tmpArr []tmpSprints = []tmpSprints{}
// 	var sprintNumber = uint(0)
// 	for startDate := time.Date(2024, time.January, 5, 0, 0, 0, 0, time.UTC); startDate.Before(time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)); startDate = startDate.Add(time.Hour * 24 * 14) {
// 		endDate := startDate.Add(time.Hour * 24 * 13)
// 		sprintNumber += 1
// 		tmpArr = append(tmpArr, tmpSprints{Number: sprintNumber, StartDate: startDate, EndDate: endDate})
// 		t := time.Now()
// 		if (t.Equal(startDate) || t.After(startDate)) && (t.Equal(endDate) || t.Before(endDate)) {
// 			fmt.Printf("YES: %d", sprintNumber)
// 		}
// 	}
// 	printResp(tmpArr)
// 	return tmpArr
// }










package main

import (
	"encoding/json"
	"fmt"
	"time"

	ops "Users/alexeylychkin/Desktop/NeoToolsBackend/draft/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var Total = 0

func main() {
	fmt.Println("Hi!")

	db, err := gorm.Open(sqlite.Open("gorm.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	//https://gorm.io/docs/migration.html
	db.Migrator().DropTable(
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

	ops.InitUnionOne()
	computeAutoOperations()
}

func printResp(model any) {
	queryResp, _ := json.MarshalIndent(model, "", "  ")
	fmt.Println(string(queryResp))
}

func printM2AResp(AOper tmpAutoOperation, msg string) {
	fmt.Println("------")
	fmt.Println(string(AOper.FundLabel))
	fmt.Println(string(AOper.FundID))
	fmt.Printf(
		"check: %v | у родителя %d| в фонде: %d | передает детям: %d | вернул ребенок в фонд: %d | общий баланс: %d \n",
		AOper.CheckChild, AOper.ParentValue, AOper.FundValue, AOper.ToChild, AOper.FromChild, AOper.Balance,
	)
	fmt.Println(string(msg))
	fmt.Println("------")
}

// func printDebugResp(model any, parentValue int, sentToFund int, balance int, rootValue int) {
// 	queryResp, _ := json.MarshalIndent(model, "", "  ")
// 	fmt.Println("------")
// 	fmt.Println(string(queryResp))
// 	fmt.Printf("parent: %d | current: %d | balance: %d| root: %d \n", parentValue, sentToFund, balance, rootValue)
// 	fmt.Println("------")
// }

// type tmpFundName struct {
// 	ID    string
// 	Label string
// }

type tmpManual struct {
	ID     string
	Value  int
	FundID string
}

type tmpAutoOperation struct {
	ParentValue int  //Сколько у родителя в фонде
	FundValue   int  //Сколько сейчас в фонде
	CheckChild  bool //Собираем ли фонд из детей? Да - FundValue = parentValue
	ToChild     int  //Сколько отправлено на распределение в детей
	FromChild   int  //Сколько накопилось из детей
	Balance     int  //Какой стал баланс после распределния в фонд
	PrevFund    string
	FundID      uint
	FundLabel   string
}

// https://gorm.io/docs/query.html#Struct-amp-Map-Conditions
func computeAutoOperations() {

	db, err := gorm.Open(sqlite.Open("gorm.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	//Получаем мануальную операцию, по которой нужно сгенерировать автоматические
	var manualOperation ops.ManualOperation
	var manual tmpManual
	db.First(&manualOperation, 1).Scan(&manual)
	fmt.Println("-----------------")
	fmt.Println("Manual operation:")
	printResp(manual)
	fmt.Println("-----------------")

	var fund ops.Fund
	db.Model(&ops.Fund{}).Preload("Child").Preload("Goals").First(&fund, manual.FundID)
	// printResp(fund)

	manualToAutoPayment := tmpAutoOperation{
		ParentValue: manual.Value,
		FundValue:   manual.Value,
		CheckChild:  fund.CheckChild,
		ToChild:     manual.Value * fund.RuleValue / 100,
		FromChild:   0,
		Balance:     manual.Value,
		PrevFund:    "Мануальная операция",
		FundID:      fund.ID,
		FundLabel:   fund.Label,
	}
	// printResp(manualToAutoPayment)

	M2A(fund, db, manualToAutoPayment)
}

//
// RuleValue = процент, который от родительского фонда / операции (!= null) распределяется по детям
// Если RuleValue = -1 и CheckChild = true, вычисляется от детей. То есть сумма их значений
// Если в детях есть правило %, то смотрится родитель фонда, у которого был CheckChild = true
// ___________________
// Если RuleValue = -1 и CheckChild = false и нет Goals, вычисляется по остаточному принципу
// Если RuleValue = -1 и CheckChild = false и есть Goals, вычисляется по целям
// Если RuleValue > 0 и CheckChild = false, то от заданного RuleValue, рассчитывается значение детей
// ___________________
// Если RuleValue = 0. то в фонд НЕ идет отчислений, проверку детей - пропускаем
// ___________________
// Деньги на цели отправляются в их родительских фонд, далее пропорционально распределяются по целям в рамках Бизнес-логики

func M2A(fund ops.Fund, db *gorm.DB, operation tmpAutoOperation) tmpAutoOperation {
	db.Model(&ops.Fund{}).Preload("Child").Preload("Goals").First(&fund, fund.ID)
	// fmt.Println("\n<== " + fund.Label + " ==>")

	if fund.RuleValue != 0 {
		if len(fund.Child) > 0 {
			if fund.CheckChild {
				OperationChecker := operation
				for _, child := range fund.Child {
					OperationChecker.PrevFund = operation.FundLabel
					OperationChecker.CheckChild = child.CheckChild
					OperationChecker.FundID = child.ID
					OperationChecker.FundLabel = child.Label
					operation.FromChild += M2A(*child, db, OperationChecker).FundValue
				}
				operation.FundValue = operation.FromChild
				operation.Balance = operation.Balance - operation.FundValue
			} else { // Если фонд самодостаточный (НЕ зависит от детей)
				if fund.RuleValue < 0 { //Если сумма рассчитывается по остаточному принципу (Чистая выручка)
					operation.ParentValue = operation.Balance //Тк это остаточный принцип, то он теперь родитель
					operation.ToChild = operation.Balance
				} else {
					operation.ToChild = operation.ParentValue * fund.RuleValue / 100
				}
				operation.FundValue = operation.ToChild //Мы знаем значение фонда, тк от него и рассчитывается

				OperationToChild := operation
				//Если родитель чекает детей, то отталкиваемся от его родителя
				//Если передает детям, то считаем от передачи
				if !operation.CheckChild {
					OperationToChild.ParentValue = operation.ToChild
				}
				OperationToChild.FundValue = 0
				OperationToChild.ToChild = 0
				OperationToChild.FromChild = 0
				for _, child := range fund.Child {
					OperationToChild.PrevFund = operation.FundLabel
					OperationToChild.CheckChild = child.CheckChild
					OperationToChild.FundID = child.ID
					OperationToChild.FundLabel = child.Label

					operation.Balance -= M2A(*child, db, OperationToChild).FundValue
					OperationToChild.Balance = operation.Balance
				}
			}
		} else { //ЕСЛИ ЭТО ОКОНЕЧНЫЙ ФОНД (нет детей)
			if len(fund.Goals) > 0 {
				for _, goal := range fund.Goals {
					operation.FundValue += (int(goal.Total) / (int(time.Until(goal.ExpireDate).Hours() / 24)) * 14)
				}
			} else {
				operation.FundValue = operation.ParentValue * fund.RuleValue / 100
			}
			// fmt.Println("ЕСЛИ ЭТО ОКОНЕЧНЫЙ ФОНД (нет детей)")
		}
		printM2AResp(operation, "Все ок")
		autoOper := ops.AutoOperation{
			Date:              time.Now(),
			Value:             uint(operation.FundValue),
			ManualOperationID: "1",
			FundID:            string(operation.FundID),
			OperationStatusID: "999",
		}
		db.Create(&autoOper) // pass pointer of data to Create
	}

	return operation
	// if fund.RuleValue != 0 {
	// 	if len(fund.Child) > 0 {
	// 		if fund.CheckChild {
	// 			for _, v := range fund.Child {
	// 				if v.CheckChild {
	// 					//TODO: добавить проверку детей на проверку детей
	// 					fmt.Println("Есть проверка детей у чекаря детей")
	// 				} else {
	// 					operation.FundValue += M2A(*v, db, tmpAutoOperation{
	// 						FundValue:  operation.FundValue * v.RuleValue / 100, //mb 0
	// 						CheckChild: fund.CheckChild,
	// 						ToChild:    operation.FundValue * v.RuleValue / 100,
	// 						FromChild:  0,
	// 						Balance:    operation.Balance,
	// 						PrevFund:   fund.Label,
	// 						FundID:     v.ID,
	// 						FundLabel:  v.Label,
	// 					}).FundValue //МБ from child
	// 					operation.Balance -= operation.FundValue
	// 				}
	// 			}
	// 			operation.FundValue = operation.FromChild
	// 			operation.Balance = operation.Balance - operation.FundValue

	// 			// printM2AResp(checkChild, operation, "Ваще-то собираем с детей") // TODO: отправить наверх
	// 		} else { //fund.CheckChild == false
	// 			for _, v := range fund.Child {
	// 				if v.CheckChild {
	// 					operation.ToChild += M2A(*v, db, tmpAutoOperation{
	// 						FundValue:  operation.ToChild,
	// 						CheckChild: v.CheckChild,
	// 						ToChild:    0,
	// 						FromChild:  0,
	// 						Balance:    operation.Balance,
	// 						PrevFund:   fund.Label,
	// 						FundID:     v.ID,
	// 						FundLabel:  v.Label,
	// 					}).FundValue
	// 					operation.Balance -= operation.FundValue
	// 					// operation.FundValue = operation.FromChild
	// 					// operation.Balance = operation.Balance - operation.FundValue
	// 				} else {
	// 					fmt.Println("ELSE CheckChild 1")
	// 				}
	// 			}
	// 			fmt.Println("ELSE CheckChild")
	// 			for _, v := range fund.Child {
	// 				if v.RuleValue != 0 {
	// 					if fund.RuleValue > 0 {
	// 						//!!! operation =
	// 						// fmt.Println(v.RuleValue)
	// 						// fmt.Println(operation.FundValue)

	// 						// Вернул выручке Баланс
	// 						operation.Balance = M2A(*v, db, tmpAutoOperation{
	// 							FundValue:  operation.FundValue * v.RuleValue / 100,
	// 							CheckChild: fund.CheckChild,
	// 							ToChild:    operation.FundValue * v.RuleValue / 100,
	// 							FromChild:  0,
	// 							Balance:    operation.Balance,
	// 							PrevFund:   fund.Label,
	// 							FundID:     v.ID,
	// 							FundLabel:  v.Label,
	// 						}).Balance
	// 						// operation.Balance = operation.ParentValue - operation.FundValue // ParentValue под сомнением

	// 					} else {
	// 						fmt.Println("ELSE CheckChild: RuleValue <= 0")
	// 						fmt.Print("\n\n\n\n\nЧИСТАЯ ВЫРУЧКА \n\n\n\n\n")
	// 						//УБРАТЬ .FundValue?? НА ССО Балан неверный // Путаница баланса и ToChild
	// 						operation.ToChild = M2A(*v, db, tmpAutoOperation{
	// 							FundValue:  operation.Balance,
	// 							CheckChild: fund.CheckChild,
	// 							ToChild:    operation.Balance,
	// 							FromChild:  0,
	// 							Balance:    operation.Balance,
	// 							PrevFund:   fund.Label,
	// 							FundID:     v.ID,
	// 							FundLabel:  v.Label,
	// 						}).ToChild
	// 						operation.FundValue = operation.ToChild
	// 					} //ЕСЛИ < 0, то передаем только остаток баланса
	// 				}
	// 			}

	// 		}
	// 	} else {
	// 		fmt.Println("ELSE Child ARRAY")
	// 		if len(fund.Goals) > 0 {
	// 			for _, goal := range fund.Goals {
	// 				operation.FundValue += (int(goal.Total) / (int(time.Until(goal.ExpireDate).Hours() / 24)) * 14)
	// 			}
	// 			operation.Balance = operation.Balance - operation.FundValue
	// 		} else {
	// 			if operation.CheckChild {
	// 				operation.FundValue = operation.FundValue * fund.RuleValue / 100
	// 				operation.Balance = operation.Balance - operation.FundValue
	// 				Total += operation.FundValue
	// 			} else {
	// 				// operation.FundValue = operation.FundValue * fund.RuleValue / 100
	// 				operation.ToChild = 0
	// 				operation.FromChild = 0
	// 				Total += operation.FundValue
	// 			}
	// 		}
	// 	}
	// 	printM2AResp(operation, "Все ок")
	// }

}



































func tpmAlg(fund ops.Fund, db *gorm.DB, checkChild bool, operation tmpAutoOperation) tmpAutoOperation {
	db.Model(&ops.Fund{}).Preload("Child").Preload("Goals").First(&fund, fund.ID)
	fmt.Println("\n<== " + fund.Label + " ==>")

	if fund.RuleValue != 0 {
		if len(fund.Child) > 0 {
			if fund.CheckChild {
				checkChild = fund.CheckChild
				operation.FundValue = 0
				operation.ToChild = 0
				operation.FromChild = 0
				for _, v := range fund.Child {
					operation.FromChild += M2A(*v, db, checkChild, tmpAutoOperation{
						FundValue: 0,
						ToChild:   0,
						FromChild: 0,
						Balance:   operation.Balance - operation.FromChild,
						PrevFund:  fund.Label,
						FundID:    v.ID,
						FundLabel: v.Label,
					}).FundValue
				}
				operation.FundValue = operation.FromChild
				operation.Balance = operation.Balance - operation.FundValue

				// printM2AResp(checkChild, operation, "Ваще-то собираем с детей") // TODO: отправить наверх
			} else { //fund.CheckChild == false
				fmt.Println("ELSE CheckChild")
				for _, v := range fund.Child {
					if v.RuleValue != 0 {
						if fund.RuleValue > 0 {
							//!!! operation =
							// fmt.Println(v.RuleValue)
							// fmt.Println(operation.FundValue)

							// Вернул выручке Баланс
							operation.Balance = M2A(*v, db, checkChild, tmpAutoOperation{
								FundValue: operation.FundValue * v.RuleValue / 100,
								ToChild:   operation.FundValue * v.RuleValue / 100,
								FromChild: 0,
								Balance:   operation.Balance,
								PrevFund:  fund.Label,
								FundID:    v.ID,
								FundLabel: v.Label,
							}).Balance
							// operation.Balance = operation.ParentValue - operation.FundValue // ParentValue под сомнением

						} else {
							fmt.Println("ELSE CheckChild: RuleValue <= 0")
							fmt.Print("\n\n\n\n\nЧИСТАЯ ВЫРУЧКА \n\n\n\n\n")
							//УБРАТЬ .FundValue?? НА ССО Балан неверный // Путаница баланса и ToChild
							operation.ToChild = M2A(*v, db, checkChild, tmpAutoOperation{
								FundValue: operation.Balance,
								ToChild:   operation.Balance,
								FromChild: 0,
								Balance:   operation.Balance,
								PrevFund:  fund.Label,
								FundID:    v.ID,
								FundLabel: v.Label,
							}).ToChild
							operation.FundValue = operation.ToChild
						} //ЕСЛИ < 0, то передаем только остаток баланса
					}
				}

			}
		} else {
			fmt.Println("ELSE Child ARRAY")
			if len(fund.Goals) > 0 {
				for _, goal := range fund.Goals {
					operation.FundValue += (int(goal.Total) / (int(time.Until(goal.ExpireDate).Hours() / 24)) * 14)
				}
				operation.Balance = operation.Balance - operation.FundValue
			} else {
				if checkChild {
					operation.FundValue = operation.FundValue * fund.RuleValue / 100
					operation.Balance = operation.Balance - operation.FundValue
					Total += operation.FundValue
				} else {
					// operation.FundValue = operation.FundValue * fund.RuleValue / 100
					operation.Balance = operation.Balance - operation.FundValue
					Total += operation.FundValue
				}
			}
		}
		printM2AResp(checkChild, operation, "Все ок")
	}

	return operation
}












func M2A(fund ops.Fund, db *gorm.DB, operation tmpAutoOperation) tmpAutoOperation {
	db.Model(&ops.Fund{}).Preload("Child").Preload("Goals").First(&fund, fund.ID)
	fmt.Println("\n<== " + fund.Label + " ==>")

	if fund.RuleValue != 0 {
		if len(fund.Child) > 0 {
			if fund.CheckChild {
				for _, v := range fund.Child {
					if v.CheckChild {
						//TODO: добавить проверку детей на проверку детей
						fmt.Println("Есть проверка детей у чекаря детей")
					} else {
						operation.FundValue += M2A(*v, db, tmpAutoOperation{
							FundValue:  operation.FundValue * v.RuleValue / 100, //mb 0
							CheckChild: fund.CheckChild,
							ToChild:    operation.FundValue * v.RuleValue / 100,
							FromChild:  0,
							Balance:    operation.Balance,
							PrevFund:   fund.Label,
							FundID:     v.ID,
							FundLabel:  v.Label,
						}).FundValue //МБ from child
						operation.Balance -= operation.FundValue
					}
				}
				operation.FundValue = operation.FromChild
				operation.Balance = operation.Balance - operation.FundValue

				// printM2AResp(checkChild, operation, "Ваще-то собираем с детей") // TODO: отправить наверх
			} else { //fund.CheckChild == false
				for _, v := range fund.Child {
					if v.CheckChild {
						operation.ToChild += M2A(*v, db, tmpAutoOperation{
							FundValue:  operation.ToChild,
							CheckChild: v.CheckChild,
							ToChild:    0,
							FromChild:  0,
							Balance:    operation.Balance,
							PrevFund:   fund.Label,
							FundID:     v.ID,
							FundLabel:  v.Label,
						}).FundValue
						operation.Balance -= operation.FundValue
						// operation.FundValue = operation.FromChild
						// operation.Balance = operation.Balance - operation.FundValue
					} else {
						fmt.Println("ELSE CheckChild 1")
					}
				}
				fmt.Println("ELSE CheckChild")
				for _, v := range fund.Child {
					if v.RuleValue != 0 {
						if fund.RuleValue > 0 {
							//!!! operation =
							// fmt.Println(v.RuleValue)
							// fmt.Println(operation.FundValue)

							// Вернул выручке Баланс
							operation.Balance = M2A(*v, db, tmpAutoOperation{
								FundValue:  operation.FundValue * v.RuleValue / 100,
								CheckChild: fund.CheckChild,
								ToChild:    operation.FundValue * v.RuleValue / 100,
								FromChild:  0,
								Balance:    operation.Balance,
								PrevFund:   fund.Label,
								FundID:     v.ID,
								FundLabel:  v.Label,
							}).Balance
							// operation.Balance = operation.ParentValue - operation.FundValue // ParentValue под сомнением

						} else {
							fmt.Println("ELSE CheckChild: RuleValue <= 0")
							fmt.Print("\n\n\n\n\nЧИСТАЯ ВЫРУЧКА \n\n\n\n\n")
							//УБРАТЬ .FundValue?? НА ССО Балан неверный // Путаница баланса и ToChild
							operation.ToChild = M2A(*v, db, tmpAutoOperation{
								FundValue:  operation.Balance,
								CheckChild: fund.CheckChild,
								ToChild:    operation.Balance,
								FromChild:  0,
								Balance:    operation.Balance,
								PrevFund:   fund.Label,
								FundID:     v.ID,
								FundLabel:  v.Label,
							}).ToChild
							operation.FundValue = operation.ToChild
						} //ЕСЛИ < 0, то передаем только остаток баланса
					}
				}

			}
		} else {
			fmt.Println("ELSE Child ARRAY")
			if len(fund.Goals) > 0 {
				for _, goal := range fund.Goals {
					operation.FundValue += (int(goal.Total) / (int(time.Until(goal.ExpireDate).Hours() / 24)) * 14)
				}
				operation.Balance = operation.Balance - operation.FundValue
			} else {
				if operation.CheckChild {
					operation.FundValue = operation.FundValue * fund.RuleValue / 100
					operation.Balance = operation.Balance - operation.FundValue
					Total += operation.FundValue
				} else {
					// operation.FundValue = operation.FundValue * fund.RuleValue / 100
					operation.ToChild = 0
					operation.FromChild = 0
					Total += operation.FundValue
				}
			}
		}
		printM2AResp(operation, "Все ок")
	}

	return operation
}























package main

import (
	"encoding/json"
	"fmt"
	"time"

	ops "Users/alexeylychkin/Desktop/NeoToolsBackend/draft/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var Total = 0

func main() {
	fmt.Println("Hi!")

	db, err := gorm.Open(sqlite.Open("gorm.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	//https://gorm.io/docs/migration.html
	db.Migrator().DropTable(
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

	ops.InitUnionOne()
	computeAutoOperations()
}

func printResp(model any) {
	queryResp, _ := json.MarshalIndent(model, "", "  ")
	fmt.Println(string(queryResp))
}

func printM2AResp(checkChild bool, AOper tmpAutoOperation, msg string) {
	fmt.Println("------")
	fmt.Println(string(AOper.FundLabel))
	fmt.Printf(
		"check: %v | у родителя: %d | в фонде: %d | отдано детям: %d | взято у детей: %d | баланс: %d \n",
		checkChild, AOper.ParentValue, AOper.FundValue, AOper.ToChild, AOper.FromChild, AOper.Balance,
	)
	fmt.Println(string(msg))
	fmt.Println("------")
}

// func printDebugResp(model any, parentValue int, sentToFund int, balance int, rootValue int) {
// 	queryResp, _ := json.MarshalIndent(model, "", "  ")
// 	fmt.Println("------")
// 	fmt.Println(string(queryResp))
// 	fmt.Printf("parent: %d | current: %d | balance: %d| root: %d \n", parentValue, sentToFund, balance, rootValue)
// 	fmt.Println("------")
// }

// type tmpFundName struct {
// 	ID    string
// 	Label string
// }

type tmpManual struct {
	ID     string
	Value  int
	FundID string
}

type tmpAutoOperation struct {
	ParentValue int //Сколько денег было в родителе
	FundValue   int //Сколько сейчас в фонде
	ToChild     int //Сколько отправлено на распределение в детей
	FromChild   int //Сколько накопилось из детей
	Balance     int //Какой стал баланс после распределния в фонд
	PrevFund    string
	FundID      uint
	FundLabel   string
}

// https://gorm.io/docs/query.html#Struct-amp-Map-Conditions
func computeAutoOperations() {

	db, err := gorm.Open(sqlite.Open("gorm.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	//Получаем мануальную операцию, по которой нужно сгенерировать автоматические
	var manualOperation ops.ManualOperation
	var manual tmpManual
	db.First(&manualOperation, 1).Scan(&manual)
	fmt.Println("-----------------")
	fmt.Println("Manual operation:")
	printResp(manual)
	fmt.Println("-----------------")

	var fund ops.Fund
	db.Model(&ops.Fund{}).Preload("Child").Preload("Goals").First(&fund, manual.FundID)
	// printResp(fund)

	manualToAutoPayment := tmpAutoOperation{
		ParentValue: manual.Value,
		FundValue:   manual.Value,
		ToChild:     manual.Value * fund.RuleValue / 100,
		FromChild:   0,
		Balance:     manual.Value,
		PrevFund:    "Мануальная операция",
		FundID:      fund.ID,
		FundLabel:   fund.Label,
	}

	M2A(fund, db, fund.CheckChild, manualToAutoPayment)

	println(Total)
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
// ___________________
// Деньги на цели отправляются в их родительских фонд, далее пропорционально распределяются по целям в рамках Бизнес-логики

func M2A(fund ops.Fund, db *gorm.DB, checkChild bool, operation tmpAutoOperation) tmpAutoOperation {
	db.Model(&ops.Fund{}).Preload("Child").Preload("Goals").First(&fund, fund.ID)
	fmt.Println("\n<== " + fund.Label + " ==>")

	if fund.RuleValue != 0 {
		if len(fund.Child) > 0 {
			if fund.CheckChild {
				checkChild = fund.CheckChild
				operation.FundValue = 0
				operation.ToChild = 0
				operation.FromChild = 0
				for _, v := range fund.Child {
					operation.FromChild += M2A(*v, db, checkChild, tmpAutoOperation{
						ParentValue: operation.ParentValue,
						FundValue:   0,
						ToChild:     0,
						FromChild:   0,
						Balance:     operation.Balance - operation.FromChild,
						PrevFund:    fund.Label,
						FundID:      v.ID,
						FundLabel:   v.Label,
					}).FundValue
				}
				operation.FundValue = operation.FromChild
				operation.Balance = operation.Balance - operation.FundValue

				// printM2AResp(checkChild, operation, "Ваще-то собираем с детей") // TODO: отправить наверх
			} else {
				fmt.Println("ELSE CheckChild")
				for _, v := range fund.Child {
					if v.RuleValue != 0 {
						if fund.RuleValue > 0 {
							//!!! operation =
							// fmt.Println(v.RuleValue)
							// fmt.Println(operation.FundValue)

							// Вернул выручке Баланс
							operation.Balance = M2A(*v, db, checkChild, tmpAutoOperation{
								ParentValue: operation.ParentValue,
								FundValue:   operation.FundValue * fund.RuleValue / 100,
								ToChild:     operation.FundValue * fund.RuleValue / 100,
								FromChild:   0,
								Balance:     operation.Balance,
								PrevFund:    fund.Label,
								FundID:      v.ID,
								FundLabel:   v.Label,
							}).Balance
							// operation.Balance = operation.ParentValue - operation.FundValue // ParentValue под сомнением

						} else {
							fmt.Println("ELSE CheckChild: RuleValue <= 0")
							fmt.Print("\n\n\n\n\nЧИСТАЯ ВЫРУЧКА \n\n\n\n\n")
							//УБРАТЬ .FundValue?? НА ССО Балан неверный // Путаница баланса и ToChild
							operation.ToChild = M2A(*v, db, checkChild, tmpAutoOperation{
								ParentValue: operation.Balance,
								FundValue:   operation.Balance,
								ToChild:     operation.Balance,
								FromChild:   0,
								Balance:     operation.Balance,
								PrevFund:    fund.Label,
								FundID:      v.ID,
								FundLabel:   v.Label,
							}).ToChild
							operation.FundValue = operation.ToChild
						} //ЕСЛИ < 0, то передаем только остаток баланса
					}
				}

			}
		} else {
			fmt.Println("ELSE Child ARRAY")
			if len(fund.Goals) > 0 {
				for _, goal := range fund.Goals {
					operation.FundValue += (int(goal.Total) / (int(time.Until(goal.ExpireDate).Hours() / 24)) * 14)
				}
				operation.Balance = operation.Balance - operation.FundValue
			} else {
				if checkChild {
					operation.FundValue = operation.ParentValue * fund.RuleValue / 100
					operation.Balance = operation.Balance - operation.FundValue
					Total += operation.FundValue
				} else {
					// operation.FundValue = operation.FundValue * fund.RuleValue / 100
					operation.Balance = operation.Balance - operation.FundValue
					Total += operation.FundValue
				}
			}
		}
		printM2AResp(checkChild, operation, "Все ок")
	}

	return operation
}




































// TODO: добавить рекурсивный обход детей в отдельной функции. Не путаем детей и родителей - собирателей в переменных баланса и сендов
func tmpFund(fund ops.Fund, db *gorm.DB, checkChild bool, parentValue int, sentToFund int, balance int, rootValue int) (int, int, int, int) {
	var fundName tmpFundName
	db.First(&fund, fund.ID).Scan(&fundName)
	printResp(fundName)

	var childFund ops.Fund
	db.Model(&ops.Fund{}).Preload("Child").Preload("Goals").First(&childFund, fund.ID)

	if fund.Label == "Чистая выручка" {
		fmt.Print("\n\n\nWHERE\n\n\n")
	}

	// println("MODEL _____________")
	// printResp(childFund)

	if fund.CheckChild {
		if len(childFund.Child) > 0 { // TODO: чекнуть НЕ -1
			if childFund.RuleValue == -1 {
				fmt.Println("Check Childs !(-1):\t\t" + fund.Label)
				parentValue = 0
				sentToFund = 0
				for _, v := range childFund.Child {
					if v.RuleValue != 0 { //TODO: не факт, что правильно работает
						parentValue, sentToFund, balance, rootValue = tmpFund(*v, db, fund.CheckChild, parentValue, sentToFund, balance, rootValue)
					}
				}
				tmpAuto := tmpAutoOperation{
					Value:     parentValue,
					FundID:    fund.ID,
					FundLabel: fundName.Label,
				}
				printDebugResp(tmpAuto, parentValue, sentToFund, balance, rootValue)
			} else if childFund.RuleValue > 0 {
				fmt.Println("Check Childs !(>0):\t\t" + fund.Label)

				//TODO: Сделать через parentValue. Если -1, то перент - это рут, если нет, то перент = процет от баланса
				// rootValue = balance * childFund.RuleValue / 100
				// tmpAuto := tmpAutoOperation{
				// 	Value:     rootValue,
				// 	FundID:    fund.ID,
				// 	FundLabel: fundName.Label,
				// }
				// printDebugResp(tmpAuto, parentValue, sentToFund, balance, rootValue)

				// for _, v := range childFund.Child {
				// 	if v.RuleValue != 0 { //TODO: не факт, что правильно работает
				// 		parentValue, sentToFund, balance, rootValue = tmpFund(*v, db, parentValue, sentToFund, balance, rootValue)
				// 	}
				// }

			}
		}
	} else { //ChechChild == False
		if fund.RuleValue > 0 { //Выручка
			fmt.Println("RuleValue > 0\t\t" + fund.Label)
			if len(childFund.Child) > 0 {
				if fund.RuleValue == 100 {
					sentToFund = rootValue
					parentValue = 0
					balance = rootValue
				} else if checkChild {
					sentToFund = rootValue * fund.RuleValue / 100
					parentValue += sentToFund
					balance -= sentToFund
				} else {
					sentToFund = parentValue * fund.RuleValue / 100
					parentValue = sentToFund
					balance -= sentToFund
					println("Check in False")
				}

				tmpAuto := tmpAutoOperation{
					Value:     sentToFund,
					FundID:    fund.ID,
					FundLabel: fundName.Label,
				}

				printDebugResp(tmpAuto, parentValue, sentToFund, balance, rootValue)

				for _, v := range childFund.Child {
					fmt.Println("Нашли ребенка: " + v.Label)
					parentValue, sentToFund, balance, rootValue = tmpFund(*v, db, checkChild, parentValue, sentToFund, balance, rootValue)
				}
			} else {
				if fund.RuleValue == 100 {
					sentToFund = rootValue
					parentValue = 0
					balance = rootValue
				} else if checkChild {
					sentToFund = rootValue * fund.RuleValue / 100
					parentValue += sentToFund
					balance -= sentToFund
				} else {
					sentToFund = parentValue * fund.RuleValue / 100
					balance -= sentToFund
					println("Check in False")
				}

				tmpAuto := tmpAutoOperation{
					Value:     sentToFund,
					FundID:    fund.ID,
					FundLabel: fundName.Label,
				}

				printDebugResp(tmpAuto, parentValue, sentToFund, balance, rootValue)
			}

			// for _, v := range childFund.Child {
			// 	if v.RuleValue != 0 { //TODO: не факт, что правильно работает
			// 		parentValue, sentToFund, balance, rootValue = tmpFund(*v, db, parentValue, sentToFund, balance, rootValue)
			// 	}
			// }

			return parentValue, sentToFund, balance, rootValue

		} else if fund.RuleValue == 0 {

			fmt.Println("Zero Value\t\t" + fund.Label)

		} else { //RuleValue == -1
			if len(childFund.Goals) > 0 {
				var sentToFund = 0

				fmt.Println("Goals List\t\t" + fund.Label)
				fmt.Println(sentToFund)
				for _, v := range childFund.Goals {
					tmpValue := (int(v.Total) / (int(time.Until(v.ExpireDate).Hours() / 24)) * 14)
					sentToFund += tmpValue
					fmt.Println(tmpValue)
				}

				if sentToFund > balance {
					tmpAuto := tmpAutoOperation{
						Value:     balance,
						FundID:    fund.ID,
						FundLabel: fundName.Label,
					}
					sentToFund = balance
					balance -= sentToFund
					parentValue += sentToFund
					printDebugResp(tmpAuto, parentValue, sentToFund, balance, rootValue)
				} else {
					tmpAuto := tmpAutoOperation{
						Value:     sentToFund,
						FundID:    fund.ID,
						FundLabel: fundName.Label,
					}
					balance -= sentToFund
					parentValue += sentToFund
					printDebugResp(tmpAuto, parentValue, sentToFund, balance, rootValue)
				}

			} else {
				fmt.Println("-1 Value, NO GOALS\t\t" + fund.Label)

				if len(childFund.Child) > 0 {
					fmt.Println("Check Related Fund Childs:\t\t" + fund.Label)
					parentValue = balance
					sentToFund = balance

					tmpAuto := tmpAutoOperation{
						Value:     sentToFund,
						FundID:    fund.ID,
						FundLabel: fundName.Label,
					}
					printDebugResp(tmpAuto, parentValue, sentToFund, balance, rootValue)

					for _, v := range childFund.Child {
						if v.RuleValue != 0 { //TODO: не факт, что правильно работает
							parentValue, sentToFund, balance, rootValue = tmpFund(*v, db, childFund.CheckChild, parentValue, sentToFund, balance, rootValue)
						}
					}

				}
			}
		}
	}

	return parentValue, sentToFund, balance, rootValue
}
















func initTypes() localTypes {
	return localTypes{
		FundTypeKingPercent: ops.FundType{
			ValueType:        "percent",
			Direction:        "king",
			LogicDescription: "Берется %, установленный в фонде, от этого рассчитывается сумма, поступающая в фонд",
		},

		FundTypeKingGoal: ops.FundType{
			ValueType:        "goal",
			Direction:        "king",
			LogicDescription: "Берется сумма, установленный в фонде, от этого рассчитывается сумма, поступающая в фонд",
		},

		FundTypeChildDepence: ops.FundType{
			ValueType:        "mixed",
			Direction:        "child",
			LogicDescription: "Сколько пойдет в фонд определяется суммами дочерних фондов",
		},

		FundTypeRest: ops.FundType{
			ValueType:        "mixed",
			Direction:        "related",
			LogicDescription: "Сумма, отходящая в фонд определяется по остаточному принципу, исходя их приоритетов других фондов",
		},

		FundTypeUEK: ops.FundType{
			ValueType:        "percent",
			Direction:        "UEK",
			LogicDescription: "Сумма, отходящая в фонд определяется по остаточному принципу, исходя их приоритетов других фондов",
		},
	}

}


func tmpFund(fund ops.Fund, db *gorm.DB, currentValue int, parentValue int, autoOperations tmpAutoOperation) tmpAutoOperation {
	var fundName tmpFundName
	db.First(&fund, fund.ID).Scan(&fundName)
	tmpAuto := tmpAutoOperation{}

	var childFund ops.Fund
	db.Model(&ops.Fund{}).Preload("Child").Preload("Goals").First(&childFund, fund.ID)
	// println("MODEL _____________")
	// printResp(childFund)

	if fund.CheckChild {
		if len(childFund.Child) > 0 { // TODO: чекнуть НЕ -1
			minusValue := 0
			for _, v := range childFund.Child {
				if v.RuleValue != 0 {
					tmpAuto = tmpFund(*v, db, currentValue-minusValue, parentValue)
					printResp(tmpAuto)
					minusValue += tmpAuto.Value
				}
			}
		}
		fmt.Println("Check Childs:\t\t" + fund.Label)
	} else { //CheckChild == False

		if fund.RuleValue > 0 { //Выручка
			tmpAuto = tmpAutoOperation{
				Value:     parentValue * fund.RuleValue / 100,
				FundID:    fund.ID,
				FundLabel: fundName.Label,
			}

			if len(fund.Child) > 0 {
				for _, v := range fund.Child {
					tmpFund(*v, db, tmpAuto.Value, tmpAuto.Value)
				}
			}

			fmt.Println("RuleValue > 0\t\t" + fund.Label)
			return tmpAuto

		} else if fund.RuleValue == 0 {

			fmt.Println("Zero Value\t\t" + fund.Label)

		} else { //RuleValue == -1
			if len(childFund.Goals) > 0 {
				var value = 0

				fmt.Println("Goals List\t\t" + fund.Label)
				fmt.Println(currentValue)
				for _, v := range childFund.Goals {
					tmpValue := (int(v.Total) / (int(time.Until(v.ExpireDate).Hours() / 24)) * 14)
					value += tmpValue
					fmt.Println(tmpValue)
				}

				if value > currentValue {
					tmpAuto = tmpAutoOperation{
						Value:     currentValue,
						FundID:    fund.ID,
						FundLabel: fundName.Label,
					}
				} else {
					tmpAuto = tmpAutoOperation{
						Value:     value,
						FundID:    fund.ID,
						FundLabel: fundName.Label,
					}
				}

			} else {
				fmt.Println("-1 Value, NO GOALS\t\t" + fund.Label)
			}
		}
	}

	return tmpAuto
}
