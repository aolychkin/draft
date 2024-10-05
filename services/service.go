package services

import (
	"Users/alexeylychkin/Desktop/NeoToolsBackend/draft/lib"
	ops "Users/alexeylychkin/Desktop/NeoToolsBackend/draft/models"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type tmpManual struct {
	ID     uint
	Date   time.Time
	Value  float32
	FundID string
}

type tmpAutoOperation struct {
	Date              time.Time
	ManualOperationID uint
	ParentValue       float32 //Сколько у родителя в фонде
	FundValue         float32 //Сколько сейчас в фонде
	CheckChild        bool    //Собираем ли фонд из детей? Да - FundValue = parentValue
	ToChild           float32 //Сколько отправлено на распределение в детей
	FromChild         float32 //Сколько накопилось из детей
	Balance           float32 //Какой стал баланс после распределния в фонд
	PrevFund          string
	FundID            uint
	FundLabel         string
}

type tmpFundTree struct {
	FundID    uint
	FundLabel string
	Value     float32
	Goals     []tmpGoalTree
	Child     []tmpFundTree
}

type tmpGoalTree struct {
	GoalId    uint
	GoalLabel string
	Value     float32
}

var Total float32 = 0.0

type cutSprint struct {
	Number    uint
	StartDate time.Time
	EndDate   time.Time
}
type cutFund struct {
	ID    uint
	Label string
}
type cutIncomeAccount struct {
	Label string
	Bank  string
}
type cutPartner struct {
	Label string
}
type cutUnion struct {
	Label string
}
type cutTeam struct {
	Label string
}
type cutOperationStatus struct {
	Type  string
	Label string
	Icon  string
	Color string
}

type frontManualOperation struct {
	ID              uint
	Date            time.Time
	Sprint          cutSprint
	Fund            cutFund
	Value           float32
	IncomeAccount   cutIncomeAccount
	Partner         cutPartner
	Union           cutUnion
	Team            cutTeam
	OperationStatus cutOperationStatus
	Details         string
}

// func ToJson(model any) {
// 	b, err := json.Marshal(model)
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// 	fmt.Println(string(b))
// }

func GetManualOperations(isPrint bool) []frontManualOperation {
	db, err := gorm.Open(sqlite.Open("gorm.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	var s2FOperation []frontManualOperation
	var manualOperations []ops.ManualOperation
	db.Find(&manualOperations).Scan(&s2FOperation)

	for iterator, manual := range manualOperations {
		db.Model(&ops.Sprint{}).Where(
			"start_date <= ? AND end_date >= ?",
			s2FOperation[iterator].Date, s2FOperation[iterator].Date).First(&s2FOperation[iterator].Sprint)

		db.First(&ops.Fund{}, manual.FundID).Scan(&s2FOperation[iterator].Fund)
		db.First(&ops.Team{}, manual.TeamID).Scan(&s2FOperation[iterator].Team)
		db.First(&ops.Union{}, manual.TeamID).Scan(&s2FOperation[iterator].Union)
		db.First(&ops.IncomeAccount{}, manual.IncomeAccountID).Scan(&s2FOperation[iterator].IncomeAccount)

		s2FOperation[iterator].Partner = cutPartner{
			Label: "ООО МЕДЛАБ ПЛЮС",
		}
		s2FOperation[iterator].OperationStatus = cutOperationStatus{
			Type:  "paid",
			Label: "Оплачен",
			Icon:  "mdi-light:home",
			Color: "success",
		}

		if isPrint {
			lib.PrintResp(s2FOperation[iterator])
		}
	}

	b, err := json.Marshal(s2FOperation)
	if err != nil {
		fmt.Println(err)
		return []frontManualOperation{}
	}

	if isPrint {
		fmt.Println(string(b))
	}

	return s2FOperation
}

func GetFundTreeByManualOperationID(manualOperationID uint, isPrint bool) tmpFundTree {
	fundTree := computeFundTreeByManualOperationID(manualOperationID)

	b, err := json.Marshal(fundTree)
	if err != nil {
		fmt.Println(err)
		return tmpFundTree{}
	}
	if isPrint {
		fmt.Println(string(b))
	}

	return fundTree
}

func computeFundTreeByManualOperationID(manualOperationID uint) tmpFundTree {
	db, err := gorm.Open(sqlite.Open("gorm.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	var manualOperation ops.ManualOperation
	db.First(&manualOperation, manualOperationID)

	var fund ops.Fund
	db.Model(&ops.Fund{}).Preload("Child").Preload("Goals").First(&fund, manualOperation.FundID)

	rootFund := tmpFundTree{
		FundID:    fund.ID,
		FundLabel: fund.Label,
		Value:     manualOperation.Value,
	}

	return ComputeFundTrees(fund, db, manualOperationID, rootFund)
}

func ComputeFundTrees(fund ops.Fund, db *gorm.DB, manualOperationID uint, fundTree tmpFundTree) tmpFundTree {
	db.Model(&ops.Fund{}).Preload("Child").Preload("Goals").First(&fund, fund.ID)
	//TODO - закинуть размер фондов по автооперациям
	if len(fund.Child) > 0 {
		for _, child := range fund.Child {
			toNext := tmpFundTree{
				FundID:    child.ID,
				FundLabel: child.Label,
				Value:     0,
			}

			if fund.RuleValue != 0 && child.RuleValue != 0 {
				var autoOperations []ops.AutoOperation

				err := db.Where(
					"manual_operation_id = ? AND fund_id = ? AND goals_id = ''",
					manualOperationID, child.ID,
				).Find(&autoOperations).Error
				if errors.Is(err, gorm.ErrRecordNotFound) {
					toNext.Value = 0
				} else {
					for _, aOperation := range autoOperations {
						toNext.Value += aOperation.Value
					}
				}
			}

			fundTree.Child = append(fundTree.Child, ComputeFundTrees(*child, db, manualOperationID, toNext))
		}
	} else {
		if len(fund.Goals) > 0 {
			for _, goal := range fund.Goals {
				if goal.Total > 0 {
					var autoOperations []ops.AutoOperation
					goalLeap := tmpGoalTree{
						GoalId:    goal.ID,
						GoalLabel: goal.Label,
						Value:     0,
					}

					err := db.Where(&ops.AutoOperation{
						ManualOperationID: strconv.FormatUint(uint64(manualOperationID), 10),
						FundID:            strconv.FormatUint(uint64(fund.ID), 10),
						GoalsID:           strconv.FormatUint(uint64(goal.ID), 10),
					}).Find(&autoOperations).Error
					if errors.Is(err, gorm.ErrRecordNotFound) {
						goalLeap.Value = 0
					} else {
						for _, aOperation := range autoOperations {
							goalLeap.Value += aOperation.Value
						}
					}

					fundTree.Goals = append(fundTree.Goals, goalLeap)
				}
			}
		}
	}

	return fundTree
}

// https://gorm.io/docs/query.html#Struct-amp-Map-Conditions
func ComputeAutoOperationsFromDB() {

	db, err := gorm.Open(sqlite.Open("gorm.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	//Получаем мануальную операцию, по которой нужно сгенерировать автоматические
	var manualModel []ops.ManualOperation
	var manualOperations []tmpManual
	db.Find(&manualModel).Scan(&manualOperations)

	for _, manual := range manualOperations {
		Total = 0
		fmt.Println("-----------------")
		fmt.Println("Manual operation:")
		lib.PrintResp(manual)
		fmt.Println("-----------------")

		//Создаем Автоматические опреации из Ручной операции в фонд
		var fund ops.Fund
		db.Model(&ops.Fund{}).Preload("Child").Preload("Goals").First(&fund, manual.FundID)

		manualToAutoPayment := tmpAutoOperation{
			Date:              manual.Date,
			ManualOperationID: manual.ID,
			ParentValue:       manual.Value,
			FundValue:         manual.Value,
			CheckChild:        fund.CheckChild,
			ToChild:           manual.Value * fund.RuleValue / 100,
			FromChild:         0,
			Balance:           manual.Value,
			PrevFund:          "Мануальная операция",
			FundID:            fund.ID,
			FundLabel:         fund.Label,
		}

		M2A(fund, db, manualToAutoPayment)
		fmt.Printf("Учтено в детях: %.2f \n", Total)
	}
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
func M2A(fund ops.Fund, db *gorm.DB, operation tmpAutoOperation) tmpAutoOperation {
	db.Model(&ops.Fund{}).Preload("Child").Preload("Goals").First(&fund, fund.ID)

	if fund.RuleValue != 0 && operation.Balance > 0 {
		if len(fund.Child) > 0 {
			if fund.CheckChild {
				OperationChecker := operation
				for _, child := range fund.Child {
					OperationChecker.PrevFund = operation.FundLabel
					OperationChecker.CheckChild = child.CheckChild
					OperationChecker.FundID = child.ID
					OperationChecker.FundLabel = child.Label

					gettedFromChild := M2A(*child, db, OperationChecker).FundValue
					operation.FromChild += gettedFromChild
					operation.Balance -= gettedFromChild
					OperationChecker.Balance = operation.Balance
				}
				operation.FundValue = operation.FromChild
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
				var goalTotalNeedsInSprint float32 = 0

				var sprint ops.Sprint
				db.Where("start_date <= ? AND end_date >= ?", operation.Date, operation.Date).First(&sprint)

				// var manualModel []ops.ManualOperation
				// var manualOperations []tmpManual
				// db.Find(&manualModel).Scan(&manualOperations)

				// Узнаем, сколько нужно пополнить за этот спринт, чтобы закрыть цель
				for _, goal := range fund.Goals {
					goalTotalNeedsInSprint += GetGoalNeedsInThisSprint(goal, db, sprint)
				}

				//Проверяем можем ли мы покрыть отведенную сумму на цели в этом спринте
				// fmt.Printf("total goal Needs: %0.2f \n", goalTotalNeeds)
				if goalTotalNeedsInSprint >= operation.Balance {
					operation.FundValue = operation.Balance
				} else {
					operation.FundValue = goalTotalNeedsInSprint
				}

				//Начисляем возможную сумму на цели, если это в принципе нужно
				if goalTotalNeedsInSprint > 0 {
					for _, goal := range fund.Goals {
						goalCoef := GetGoalNeedsInThisSprint(goal, db, sprint) / goalTotalNeedsInSprint
						goalOper := ops.AutoOperation{
							Date:              operation.Date,
							Value:             operation.FundValue * goalCoef,
							ManualOperationID: strconv.FormatUint(uint64(operation.ManualOperationID), 10),
							FundID:            strconv.FormatUint(uint64(operation.FundID), 10),
							GoalsID:           strconv.FormatUint(uint64(goal.ID), 10),
							OperationStatusID: "999",
						}
						db.Create(&goalOper)
					}
				}
			} else {
				operation.FundValue = operation.ParentValue * fund.RuleValue / 100
				operation.Balance = operation.Balance - operation.FundValue
			}

			Total += operation.FundValue // Проверяем сходимость
		}
		// printM2AResp(operation, "")
		autoOper := ops.AutoOperation{
			Date:              operation.Date,
			Value:             operation.FundValue,
			ManualOperationID: strconv.FormatUint(uint64(operation.ManualOperationID), 10),
			FundID:            strconv.FormatUint(uint64(operation.FundID), 10),
			OperationStatusID: "999",
		}
		db.Create(&autoOper)
	}

	return operation
}

// Получаем прошлые пополнения целей
func getGoalNeeds(goal ops.Goals, db *gorm.DB, sprint ops.Sprint) float32 {
	var goalAutoOperationsBeforeThisSprint []ops.AutoOperation

	db.Where(
		"date < ? AND goals_id = ?",
		sprint.StartDate, goal.ID,
	).Find(&goalAutoOperationsBeforeThisSprint)

	var goalNeeds float32 = goal.Total
	for _, v := range goalAutoOperationsBeforeThisSprint {
		goalNeeds -= v.Value
	}

	return goalNeeds
}

// Определяем, сколько денег осталось пополнить на этом спринте
func GetGoalNeedsInThisSprint(goal ops.Goals, db *gorm.DB, sprint ops.Sprint) float32 {
	var goalAutoOperationsOnThisSprint []ops.AutoOperation
	db.Where(
		"date >= ? AND date <= ? AND goals_id = ?",
		sprint.StartDate, sprint.EndDate, int(goal.ID),
	).Find(&goalAutoOperationsOnThisSprint)

	//Считаем, на сколько уже была выполнена цель
	var goalAlreadyIncomesOnThisSprint float32 = 0
	for _, gAutoOperValue := range goalAutoOperationsOnThisSprint {
		goalAlreadyIncomesOnThisSprint += gAutoOperValue.Value
	}

	goalNeeds := getGoalNeeds(goal, db, sprint)

	return ((goalNeeds / float32(goal.ExpireDate.Sub(sprint.StartDate).Hours()/24)) * 14) - goalAlreadyIncomesOnThisSprint
} //Протестить функцию
