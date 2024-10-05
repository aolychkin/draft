package models

import (
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func initSprints() []Sprint {
	var tmpArr []Sprint = []Sprint{}
	var sprintNumber = uint(0)
	for startDate := time.Date(2024, time.January, 5, 0, 0, 0, 0, time.UTC); startDate.Before(time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)); startDate = startDate.Add(time.Hour * 24 * 14) {
		endDate := startDate.Add(time.Hour*24*13 + 23*time.Hour + 59*time.Minute + 59*time.Second)
		sprintNumber += 1
		tmpArr = append(tmpArr, Sprint{Number: sprintNumber, StartDate: startDate, EndDate: endDate})
	}
	return tmpArr
}

// https://gorm.io/docs/create.html
// ЦЕЛИ по ЗП рассчитываются в отдельной вкладке на спринт каждый спринт
// ИЛИ по ДЖОБЕ!)
func InitUnionOne() {
	db, err := gorm.Open(sqlite.Open("gorm.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	unionOne := []Union{{
		Label: "Founder Union",
		IncomeAccount: []IncomeAccount{
			{Label: "ИП Лычкин А.О.", Bank: "Точка Банк"},
		},
		Team: []Team{
			{
				Label:  "ГК. HR-партнерка",
				Sprint: initSprints(),
			},
		},
	}}

	//Сначала расчитывается фонд с наивысшим приоритетом, затем от остатка родителя - остальные
	db.Create(&Fund{
		Label:    "Выручка",
		Priority: 0,
		Union:    unionOne,
		ManualOperation: []ManualOperation{
			{
				Date:            time.Now().Add(-28 * 24 * time.Hour),
				Value:           1000,
				TeamID:          "1",
				IncomeAccountID: "1",
				Details:         "Test operation 1",
			},
			{
				Date:            time.Now(),
				Value:           3000,
				TeamID:          "1",
				IncomeAccountID: "1",
				Details:         "Test operation 2",
			},
			{
				Date:            time.Now(),
				Value:           100000,
				TeamID:          "1",
				IncomeAccountID: "1",
				Details:         "Test operation 3",
			},
		},
		CheckChild: false,
		RuleValue:  100,
		Child: []*Fund{
			{
				Label:      "Очищение выручки",
				Priority:   1,
				Union:      unionOne,
				CheckChild: true,
				RuleValue:  -1,
				Child: []*Fund{
					{Label: "Налог на выручку",
						Priority:   0,
						Union:      unionOne,
						CheckChild: false,
						RuleValue:  6, //TODO: поставить 0
					},
					{Label: "Обязательный маркетинг",
						Priority:   0,
						Union:      unionOne,
						CheckChild: false,
						RuleValue:  25,
					},
					{Label: "Прямые переменные расходы",
						Priority:   0,
						Union:      unionOne,
						CheckChild: false,
						RuleValue:  0,
					},
					{Label: "Возврат инвестиций",
						Priority:   0,
						Union:      unionOne,
						CheckChild: false,
						RuleValue:  -1,
						Goals: []Goals{
							{
								Label:      "Оплата Google WorkSpace",
								Total:      18000,
								ExpireDate: time.Date(2025, time.September, 24, 00, 00, 00, 00, time.UTC),
							},
							{
								Label:      "Годовая бухгалтерия",
								Total:      11900,
								ExpireDate: time.Date(2025, time.September, 13, 00, 00, 00, 00, time.UTC),
							},
							{
								Label:      "Патент",
								Total:      19200,
								ExpireDate: time.Date(2025, time.January, 01, 00, 00, 00, 00, time.UTC),
							},
						}},
				},
			},
			{
				Label:      "Группа Фондов Стабилизации",
				Priority:   2,
				Union:      unionOne,
				CheckChild: true,
				RuleValue:  -1,
				Child: []*Fund{
					{
						Label: "Фикс Клиенских команд", Priority: 0,
						Union:      unionOne,
						CheckChild: false,
						RuleValue:  -1,
						Goals: []Goals{
							{
								Label:      "Оклад КК 1",
								Total:      0,
								ExpireDate: time.Now().Add(time.Hour * 24 * 14),
							},
						}},
					{
						Label:      "Фикс Сервисных команд",
						Priority:   0,
						Union:      unionOne,
						CheckChild: false,
						RuleValue:  -1,
						Goals: []Goals{
							{
								Label:      "Оклад СК 1",
								Total:      0,
								ExpireDate: time.Now().Add(time.Hour * 24 * 14),
							},
						}},
					{
						Label:      "Фикс Капитану",
						Priority:   0,
						Union:      unionOne,
						CheckChild: false,
						RuleValue:  -1,

						Goals: []Goals{
							{
								Label:      "Оклад Капитану О1",
								Total:      0,
								ExpireDate: time.Now().Add(time.Hour * 24 * 14),
							},
						}},
					{
						Label:      "Регулярные платежи",
						Priority:   0,
						Union:      unionOne,
						CheckChild: false,
						RuleValue:  -1,
						Goals: []Goals{
							{
								Label:      "Квант",
								Total:      998,
								ExpireDate: time.Date(2025, time.September, 24, 00, 00, 00, 00, time.UTC),
							},
							{
								Label:      "Счет Тинькофф",
								Total:      490,
								ExpireDate: time.Date(2025, time.September, 24, 00, 00, 00, 00, time.UTC),
							},
						}},

					//TODO - как оплачивать за клиентов?
					{
						Label:      "Известные платежи",
						Priority:   0,
						Union:      unionOne,
						CheckChild: false,
						RuleValue:  -1,

						Goals: []Goals{
							{
								Label:      "Страховые взносы",
								Total:      53658,
								ExpireDate: time.Date(2025, time.December, 01, 00, 00, 00, 00, time.UTC),
							},
						}},
				},
			},
			//Приоритет 3 и тип FundTypeRest, значит возможно чистой выручки не будет, так как все ушло в Стабилизацию и Очищение
			{
				Label: "Чистая выручка", Priority: 3,
				Union:      unionOne,
				CheckChild: false,
				RuleValue:  -1,
				Child: []*Fund{
					{
						Label:      "На ОО (Общее Объединения)",
						Priority:   0,
						Union:      unionOne,
						CheckChild: false,
						RuleValue:  40, // = 40% от Чистой выручки распределяется на детей
						Child: []*Fund{
							{
								Label:      "Группа Фондов Устойчивости",
								Priority:   0,
								Union:      unionOne,
								CheckChild: false,
								RuleValue:  4,
								Child: []*Fund{
									{Label: "Безопасности",
										Priority:   0,
										Union:      unionOne,
										CheckChild: false,
										RuleValue:  25,
									},
									{Label: "Тушения пожаров",
										Priority:   0,
										Union:      unionOne,
										CheckChild: false,
										RuleValue:  75, //TODO - сделать проверку на 100% распределений
									},
								},
							},
							{
								Label:      "Группа Фондов Бонусов",
								Priority:   0,
								Union:      unionOne,
								CheckChild: false,
								RuleValue:  96,
								Child: []*Fund{
									{Label: "Бонусы Клиентским командам",
										Priority:   0,
										Union:      unionOne,
										CheckChild: false,
										RuleValue:  80,
									},
									{Label: "Бонусы Сервисным командам",
										Priority:   0,
										Union:      unionOne,
										CheckChild: false,
										RuleValue:  20,
									},
								},
							},
							{
								Label:      "Группа Фондов Достигаторов",
								Priority:   0,
								Union:      unionOne,
								CheckChild: false,
								RuleValue:  0,
								Child: []*Fund{
									{Label: "Личностные игры",
										Priority:   0,
										Union:      unionOne,
										CheckChild: false,
										RuleValue:  0,
									},
									{Label: "Квартальных премий по целям",
										Priority:   0,
										Union:      unionOne,
										CheckChild: false,
										RuleValue:  0,
									},
									{Label: "Бонусы по целям Команды",
										Priority:   0,
										Union:      unionOne,
										CheckChild: false,
										RuleValue:  0,
									},
								},
							},
							{
								Label:      "Группа Фондов Социально-психологические",
								Priority:   0,
								Union:      unionOne,
								CheckChild: false,
								RuleValue:  0,
								Child: []*Fund{
									{
										Label:      "Высокий уровень энергии",
										Priority:   0,
										Union:      unionOne,
										CheckChild: false,
										RuleValue:  0,
										Child: []*Fund{
											{Label: "ДМС",
												Priority:   0,
												Union:      unionOne,
												CheckChild: false,
												RuleValue:  0, //Будет по целям
											},
											{Label: "Плюшки Спорт, Обеды, Англ",
												Priority:   0,
												Union:      unionOne,
												CheckChild: false,
												RuleValue:  0, //Будет по целям
											},
											{Label: "Эвенты, тимбилдинг",
												Priority:   0,
												Union:      unionOne,
												CheckChild: false,
												RuleValue:  0,
											},
										},
									},
									{
										Label: "Развитие таланта", Priority: 0,
										Union:      unionOne,
										CheckChild: false,
										RuleValue:  0,
										Child: []*Fund{
											{Label: "Обучение и развитие",
												Priority:   0,
												Union:      unionOne,
												CheckChild: false,
												RuleValue:  0,
											},
											{Label: "Тренинги",
												Priority:   0,
												Union:      unionOne,
												CheckChild: false,
												RuleValue:  0,
											},
										},
									},
									{Label: "Спокойствие за разрешение внеплановой ситуации",
										Priority:   0,
										Union:      unionOne,
										CheckChild: false,
										RuleValue:  0,
										Child: []*Fund{
											{Label: "Свободных финансов",
												Priority:   0,
												Union:      unionOne,
												CheckChild: false,
												RuleValue:  0,
											},
										},
									},
								},
							},
						},
					},
					{
						Label: "На ССО (Стратегическом Совете Объединения)", Priority: 0,
						Union:      unionOne,
						CheckChild: false,
						RuleValue:  20,
						Child: []*Fund{
							{Label: "Реинвестирование",
								Priority:   0,
								Union:      unionOne,
								CheckChild: false,
								RuleValue:  40,
							},
							{Label: "Всем Лидерам поровну",
								Priority:   0,
								Union:      unionOne,
								CheckChild: false,
								RuleValue:  40,
							},
							{Label: "Со всего объединения Капитану",
								Priority:   0,
								Union:      unionOne,
								CheckChild: false,
								RuleValue:  20,
							},
						},
					},
					{
						Label: "На СК (Совете Капитанов)", Priority: 0,
						Union:      unionOne,
						CheckChild: false,
						RuleValue:  40,
						Child: []*Fund{
							{Label: "Реинвестирование",
								Priority:   0,
								Union:      unionOne,
								CheckChild: false,
								RuleValue:  20,
							},
							{Label: "Со всего Общества Капитанам поровну",
								Priority:   0,
								Union:      unionOne,
								CheckChild: false,
								RuleValue:  20,
							},
							UEKData(60),
						},
					},
				},
			},
		},
	},
	)
}
