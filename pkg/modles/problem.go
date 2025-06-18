package modles

import "gorm.io/gorm"

type Example struct {
	gorm.Model
	ProblemID      uint            `json:"problem_id" gorm:"index"`
	Input          string          `json:"input"`
	ExpectedOutput string          `json:"expected_output"`
	Problem        *ProblemPropaty `json:"problem,omitempty" gorm:"foreignKey:ProblemID"`
}

type ProblemPropaty struct {
	gorm.Model
	Title       string            `json:"title"`
	Description string            `json:"description"`
	HaderFile   string            `json:"hader_file"`
	FuncBody    string            `json:"func_body"`
	MainFunc    string            `json:"main_func"`
	TestCases   []TestCaesPropaty `json:"test_cases" gorm:"foreignKey:ProblemID"`
	Difficulty  string            `json:"difficulty"` // "easy", "medium", "hard"
	Examples    []Example         `json:"examples" gorm:"foreignKey:ProblemID"`
}

type TestCaesPropaty struct {
	gorm.Model
	Input          string          `json:"input"`
	ExpectedOutput string          `json:"expected_output"`
	ProblemID      uint            `json:"problem_id" gorm:"index"`
	Problem        *ProblemPropaty `json:"problem,omitempty" gorm:"foreignKey:ProblemID"`
}

/*
docker-compose down -v    # removes containers + associated volumes
docker-compose build      # rebuilds all ima
docker-compose up --build

*/
