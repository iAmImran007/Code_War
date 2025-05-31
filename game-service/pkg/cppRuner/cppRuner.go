package cppruner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/iAmImran007/game-service/pkg/database"
	"github.com/iAmImran007/game-service/pkg/modles"
)

type JudgeResult struct {
	Passed      int   `json:"passed"`
	Total       int   `json:"total"`
	FailedCases []int `json:"failed_cases"`
}

type TestCase struct {
	Input          string `json:"input"`
	ExpectedOutput string `json:"expected_output"`
}
func JudgeCode(problemId uint, code string, testCases []TestCase, db *database.Databse) (JudgeResult, error) {
	// Create temp directory for this submission
	tempDir, err := os.MkdirTemp("", "submission_*")
	if err != nil {
		return JudgeResult{}, fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up when done

	// Define file paths
	codeFile := filepath.Join(tempDir, "submission.cpp")
	binFile := "submission.out"
	inputFile := filepath.Join(tempDir, "input.txt")
	outputFile := filepath.Join(tempDir, "output.txt")

	// Fetch header file and main func using cache
	var problem *modles.ProblemPropaty

	if db.Cache != nil {
		// Use cache
		problem, err = db.Cache.GetProblemById(problemId)
	} else {
		// Fallback to direct DB query
		var p modles.ProblemPropaty
		err = db.Db.Select("hader_file", "main_func").Where("id = ?", problemId).First(&p).Error
		problem = &p
	}

	if err != nil {
		return JudgeResult{}, fmt.Errorf("failed to fetch main and header file: %v", err)
	}

	// Join the header + user code + main file in the code
	fullCode := problem.HaderFile + "\n" + code + "\n" + problem.MainFunc

	err = os.WriteFile(codeFile, []byte(fullCode), 0644)
	if err != nil {
		return JudgeResult{}, fmt.Errorf("failed to write the full code in the c++ file: %v", err)
	}

	// Compile inside Docker
	fmt.Println("Compiling code...")
	// First, pull the gcc image if it doesn't exist
	pullCmd := exec.Command("docker", "pull", "gcc:latest")
	pullCmd.Run() // Ignore errors here, it might already exist
	
	// Compile the code
	cmd := exec.Command("docker", "run", "--rm",
		"-v", fmt.Sprintf("%s:/code", tempDir),
		"-w", "/code",
		"gcc:latest",
		"g++", "-o", binFile, "submission.cpp", "-std=c++17")

	compileOutput, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Docker command failed: %v\n", err)
		fmt.Printf("Compile output: %s\n", string(compileOutput))
		return JudgeResult{}, fmt.Errorf("compilation failed: %s", string(compileOutput))
	}
	passed := 0
	var failedCases []int

	fmt.Printf("Running %d test cases...\n", len(testCases))
	for i, tc := range testCases {
		// Write input to file
		err = os.WriteFile(inputFile, []byte(tc.Input), 0644)
		if err != nil {
			return JudgeResult{}, fmt.Errorf("failed to write input file: %v", err)
		}

		fmt.Printf("Running test case %d...\n", i+1)
		// Run binary in Docker with input
		cmd := exec.Command("docker", "run", "--rm",
			"-v", fmt.Sprintf("%s:/code", tempDir),
			"-w", "/code",
			"--memory=128m", // Set memory limit
			"--cpus=0.5",    // Set CPU limit
			"gcc:latest",
			"timeout", "2", "sh", "-c", fmt.Sprintf("./%s < input.txt > output.txt", binFile))

		runErr := cmd.Run()
		if runErr != nil {
			fmt.Printf("Test case %d execution error: %v\n", i+1, runErr)
			failedCases = append(failedCases, i+1)
			continue
		}

		// Read output
		output, err := os.ReadFile(outputFile)
		if err != nil {
			fmt.Printf("Failed to read output file for test case %d: %v\n", i+1, err)
			failedCases = append(failedCases, i+1)
			continue
		}

		// Compare output (trimming whitespace)
		actual := strings.TrimSpace(string(output))
		expected := strings.TrimSpace(tc.ExpectedOutput)

		if actual == expected {
			passed++
			fmt.Printf("Test case %d: PASSED\n", i+1)
		} else {
			failedCases = append(failedCases, i+1)
			fmt.Printf("Test case %d: FAILED\n", i+1)
			fmt.Printf("  Expected: '%s'\n", expected)
			fmt.Printf("  Actual  : '%s'\n", actual)
		}
	}

	return JudgeResult{
		Passed:      passed,
		Total:       len(testCases),
		FailedCases: failedCases,
	}, nil
}