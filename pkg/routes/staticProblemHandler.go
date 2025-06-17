package routes

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/iAmImran007/Code_War/pkg/auth"
	cppruner "github.com/iAmImran007/Code_War/pkg/cppRuner"
	"github.com/iAmImran007/Code_War/pkg/modles"
	"gorm.io/gorm"
)

type SubmissionRequest struct {
	Code string `json:"code"`
}

type ProblemResponse struct {
	ID          uint             `json:"id"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	FuncBody    string           `json:"func_body"`
	Examples    []modles.Example `json:"examples"` // Include examples in the response
}

type AllProblemsResponse struct {
	ID         uint   `json:"id"`
	Title      string `json:"title"`
	Difficulty string `json:"difficulty"`
}

// GetAllProblems - GET /problems (Public route)
func (r *Routes) GetAllProblems(w http.ResponseWriter, req *http.Request) {
	// Set security headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")

	// Only allow GET method
	if req.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	// Try to get from cache first (if available)
	var problems []modles.ProblemPropaty
	var err error

	if r.Db.Cache != nil {
		problems, err = r.Db.Cache.GetAllProblems()
	} else {
		// Fallback to direct DB query - include difficulty field
		err = r.Db.Db.Select("id", "title", "difficulty").Find(&problems).Error
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Failed to fetch problems",
		})
		return
	}

	// Debug log
	// fmt.Printf("Problems from DB/Cache: %+v\n", problems)

	var problemsResponse []AllProblemsResponse
	for _, problem := range problems {
		problemsResponse = append(problemsResponse, AllProblemsResponse{
			ID:         problem.ID,
			Title:      problem.Title,
			Difficulty: problem.Difficulty,
		})
	}

	// Debug log
	// fmt.Printf("Problems Response: %+v\n", problemsResponse)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Message: "Problems retrieved successfully",
		Data:    problemsResponse,
	})
}

// GetProblemById - GET /problem/{id} (Protected route)
func (r *Routes) GetProblemById(w http.ResponseWriter, req *http.Request) {
	// Set security headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")

	// Only allow GET method
	if req.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	// Get problem ID from URL path
	vars := mux.Vars(req)
	problemIDStr := vars["id"]

	if problemIDStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Problem ID is required",
		})
		return
	}

	// Convert string to uint
	problemID, err := strconv.ParseUint(problemIDStr, 10, 32)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Invalid problem ID format",
		})
		return
	}

	// Try to get from cache first (if available)
	var problem *modles.ProblemPropaty
	if r.Db.Cache != nil {
		problem, err = r.Db.Cache.GetProblemById(uint(problemID))
	} else {
		// Fallback to direct DB query
		var p modles.ProblemPropaty
		err = r.Db.Db.Preload("Examples").Where("id = ?", problemID).First(&p).Error
		problem = &p
	}

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Problem not found",
		})
		return
	}

	// Create response without sensitive data (TestCases, HaderFile, MainFunc)
	problemResponse := ProblemResponse{
		ID:          problem.ID,
		Title:       problem.Title,
		Description: problem.Description,
		FuncBody:    problem.FuncBody, // Include only the function body
		Examples:    problem.Examples, // Include examples in the response
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Message: "Problem retrieved successfully",
		Data:    problemResponse,
	})
}

// HandleSubmission - POST /problem/{id}/submit (Protected route)
func (r *Routes) HandleSubmition(w http.ResponseWriter, req *http.Request) {
	// Set security headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")

	// Only allow POST method
	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	// Check content type
	if !strings.Contains(req.Header.Get("Content-Type"), "application/json") {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Content-Type must be application/json",
		})
		return
	}

	// Get problem ID from URL path
	vars := mux.Vars(req)
	problemIDStr := vars["id"]

	if problemIDStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Problem ID is required",
		})
		return
	}

	// Convert string to uint
	problemID, err := strconv.ParseUint(problemIDStr, 10, 32)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Invalid problem ID format",
		})
		return
	}

	// Limit request body size (1MB)
	req.Body = http.MaxBytesReader(w, req.Body, 1048576)

	// Read and decode request body
	var submissionReq SubmissionRequest
	decoder := json.NewDecoder(req.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&submissionReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	// Basic validation
	if strings.TrimSpace(submissionReq.Code) == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Code cannot be empty",
		})
		return
	}

	// Check if code is too long (prevent abuse)
	if len(submissionReq.Code) > 50000 { // 50KB limit
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Code is too long (maximum 50,000 characters)",
		})
		return
	}

	// Try to get problem from cache first (if available)
	var problem *modles.ProblemPropaty
	if r.Db.Cache != nil {
		problem, err = r.Db.Cache.GetProblemById(uint(problemID))
	} else {
		// Fallback to direct DB query
		var p modles.ProblemPropaty
		err = r.Db.Db.Preload("TestCases").Where("id = ?", problemID).First(&p).Error
		problem = &p
	}

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Problem not found",
		})
		return
	}

	// Convert TestCaesPropaty to TestCase for judge function
	var testCases []cppruner.TestCase
	for _, tc := range problem.TestCases {
		testCases = append(testCases, cppruner.TestCase{
			Input:          tc.Input,
			ExpectedOutput: tc.ExpectedOutput,
		})
	}

	// Judge the code
	result, err := cppruner.JudgeCode(uint(problemID), submissionReq.Code, testCases, r.Db)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Compilation or runtime error",
			Data: map[string]interface{}{
				"error": err.Error(),
			},
		})
		return
	}

	// Determine status based on results
	status := "partial"
	message := "Some test cases failed"

	if result.Passed == result.Total {
		status = "accepted"
		message = "All test cases passed! Solution accepted."
		r.incrementSolvedProblems(req, uint(problemID))
 
	} else if result.Passed == 0 {
		status = "failed"
		message = "All test cases failed"
	}

	// Success response with results
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Message: message,
		Data: map[string]interface{}{
			"status":       status,
			"passed":       result.Passed,
			"total":        result.Total,
			"failed_cases": result.FailedCases,
			"problem_id":   problemID,
		},
	})
}


func (r *Routes) incrementSolvedProblems(req *http.Request, problemID uint) {
    // Get user ID from JWT token
    accessCookie, err := req.Cookie("access_token")
    if err != nil {
        return // No token, skip
    }
    
    if accessCookie.Value == "" {
        return // Empty token, skip
    }
    
    claims, err := auth.ValidateToken(accessCookie.Value)
    if err != nil {
        return // Invalid token, skip
    }
    
    // Check if user already solved this problem (optional - to avoid duplicate counting)
    var count int64
    r.Db.Db.Table("user_problems").Where("user_id = ? AND problem_id = ?", claims.UserID, problemID).Count(&count)
    
    if count == 0 { // Problem not solved before
        // Record the solution (you can create this table or skip this part)
        // For now, just increment the count
        r.Db.Db.Model(&modles.User{}).Where("id = ?", claims.UserID).
            Update("solved_problems", gorm.Expr("solved_problems + ?", 1))
    }
}

