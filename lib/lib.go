package lib

import (
	"encoding/json"
	"fmt"
	"time"
)

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

func PrintResp(model any) {
	queryResp, _ := json.MarshalIndent(model, "", "  ")
	fmt.Println(string(queryResp))
}

func PrintM2AResp(AOper tmpAutoOperation, msg string) {
	fmt.Println("------")
	fmt.Println(string(AOper.FundLabel))
	fmt.Printf(
		"id: %d check: %v | у родителя %.2f | в фонде: %.2f | передает детям: %.2f | вернул ребенок в фонд: %.2f | общий баланс: %.2f \n",
		AOper.FundID, AOper.CheckChild, AOper.ParentValue, AOper.FundValue, AOper.ToChild, AOper.FromChild, AOper.Balance,
	)
	if msg != "" {
		fmt.Println(string(msg))
	}
	fmt.Println("------")
}
