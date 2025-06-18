package cppruner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/iAmImran007/Code_War/pkg/database"
	"github.com/iAmImran007/Code_War/pkg/modles"
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
	fmt.Println("Starting judge process...")
	
	// Create temp directory for this submission
	tempDir, err := os.MkdirTemp("", "submission_*")
	if err != nil {
		return JudgeResult{}, fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up when done

	fmt.Printf("Created temp directory: %s\n", tempDir)

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
	fmt.Println("Generated full code, writing to file...")
	
	err = os.WriteFile(codeFile, []byte(fullCode), 0644)
	if err != nil {
		return JudgeResult{}, fmt.Errorf("failed to write the full code in the c++ file: %v", err)
	}

	// Check if Docker is available
	fmt.Println("Checking Docker availability...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	dockerCheck := exec.CommandContext(ctx, "docker", "version")
	if err := dockerCheck.Run(); err != nil {
		return JudgeResult{}, fmt.Errorf("docker is not available or not running: %v", err)
	}
	fmt.Println("Docker is available")

	// Check if gcc image exists locally
	fmt.Println("Checking for gcc:latest image...")
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	
	imageCheck := exec.CommandContext(ctx2, "docker", "images", "-q", "gcc:latest")
	output, err := imageCheck.Output()
	
	if err != nil || len(strings.TrimSpace(string(output))) == 0 {
		fmt.Println("gcc:latest image not found locally, pulling...")
		// Pull with timeout
		ctx3, cancel3 := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel3()
		
		pullCmd := exec.CommandContext(ctx3, "docker", "pull", "gcc:latest")
		pullOutput, pullErr := pullCmd.CombinedOutput()
		if pullErr != nil {
			fmt.Printf("Failed to pull gcc image: %v\n", pullErr)
			fmt.Printf("Pull output: %s\n", string(pullOutput))
			return JudgeResult{}, fmt.Errorf("failed to pull gcc image: %v", pullErr)
		}
		fmt.Println("Successfully pulled gcc:latest image")
	} else {
		fmt.Println("gcc:latest image found locally")
	}

	// Compile inside Docker with timeout
	fmt.Println("Compiling code...")
	ctx4, cancel4 := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel4()
	
	cmd := exec.CommandContext(ctx4, "docker", "run", "--rm",
		"-v", fmt.Sprintf("%s:/code", tempDir),
		"-w", "/code",
		"--memory=512m",      // Increased memory limit
		"--cpus=1.0",         // CPU limit
		"--network=none",     // No network access
		"gcc:latest",
		"g++", "-o", binFile, "submission.cpp", "-std=c++17")

	fmt.Printf("Running compilation command: %v\n", cmd.Args)
	
	compileOutput, err := cmd.CombinedOutput()
	if err != nil {
		if ctx4.Err() == context.DeadlineExceeded {
			return JudgeResult{}, fmt.Errorf("compilation timed out after 30 seconds")
		}
		fmt.Printf("Docker command failed: %v\n", err)
		fmt.Printf("Compile output: %s\n", string(compileOutput))
		return JudgeResult{}, fmt.Errorf("compilation failed: %s", string(compileOutput))
	}
	
	fmt.Println("Compilation successful!")
	
	// Check if binary was created
	binaryPath := filepath.Join(tempDir, binFile)
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return JudgeResult{}, fmt.Errorf("compiled binary not found at %s", binaryPath)
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
		
		// Run binary in Docker with input and timeout
		ctx5, cancel5 := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel5()
		
		runCmd := exec.CommandContext(ctx5, "docker", "run", "--rm",
			"-v", fmt.Sprintf("%s:/code", tempDir),
			"-w", "/code",
			"--memory=128m",      // Memory limit for execution
			"--cpus=0.5",         // CPU limit for execution
			"--network=none",     // No network access
			"gcc:latest",
			"timeout", "2", "sh", "-c", fmt.Sprintf("./%s < input.txt > output.txt", binFile))

		runErr := runCmd.Run()
		if runErr != nil {
			if ctx5.Err() == context.DeadlineExceeded {
				fmt.Printf("Test case %d: TIMEOUT (10s limit exceeded)\n", i+1)
			} else {
				fmt.Printf("Test case %d execution error: %v\n", i+1, runErr)
			}
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