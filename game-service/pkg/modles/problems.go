package modles

import "gorm.io/gorm"

type ProblemPropaty struct {
	gorm.Model
	Title       string            `json:"title"`
	Description string            `json:"description"`
	HaderFile   string            `json:"hader_file"`
	FuncBody    string            `json:"func_body"`
	MainFunc    string            `json:"main_func"`
	TestCases   []TestCaesPropaty `json:"test_cases" gorm:"foreignKey:ProblemID"`
}

type TestCaesPropaty struct {
	gorm.Model
	Input          string `json:"input"`
	ExpectedOutput string `json:"expected_output"`
	ProblemID      uint   `json:"problem_id"`
}

/*
docker-compose down -v    # removes containers + associated volumes
docker-compose build      # rebuilds all images
docker-compose up -d      # starts everything in detached mode
docker-compose logs -f    # follow logs

*/
