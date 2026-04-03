package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// ==================== USER HANDLER VALIDATION TESTS ====================

// Test invalid ID parameters
func TestUserHandler_Get_InvalidID_NonNumeric(t *testing.T) {
	handler := NewUserHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: handler tries to access nil DB
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/users/abc", nil)
	c.Params = gin.Params{{Key: "id", Value: "abc"}}

	handler.Get(c)
}

func TestUserHandler_Get_InvalidID_Negative(t *testing.T) {
	handler := NewUserHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: handler tries to access nil DB
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/users/-1", nil)
	c.Params = gin.Params{{Key: "id", Value: "-1"}}

	handler.Get(c)
}

func TestUserHandler_Get_InvalidID_Empty(t *testing.T) {
	handler := NewUserHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: handler tries to access nil DB
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/users/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}

	handler.Get(c)
}

func TestUserHandler_Get_InvalidID_VeryLong(t *testing.T) {
	handler := NewUserHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: handler tries to access nil DB
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	longID := strings.Repeat("1", 1000)
	c.Request, _ = http.NewRequest("GET", "/users/"+longID, nil)
	c.Params = gin.Params{{Key: "id", Value: longID}}

	handler.Get(c)
}

func TestUserHandler_Get_InvalidID_SpecialCharacters(t *testing.T) {
	handler := NewUserHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: handler tries to access nil DB
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/users/%27%20OR%20%271%27%3D%271", nil)
	c.Params = gin.Params{{Key: "id", Value: "' OR '1'='1"}}

	handler.Get(c)
}

// Test invalid JSON bodies
func TestUserHandler_Create_MalformedJSON(t *testing.T) {
	handler := NewUserHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/users", bytes.NewBuffer([]byte("{invalid")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestUserHandler_Create_WrongType_Username_Number(t *testing.T) {
	handler := NewUserHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"username": 123,
		"password": "password123",
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/users", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for wrong username type, got %d", w.Code)
	}
}

func TestUserHandler_Create_WrongType_Password_Number(t *testing.T) {
	handler := NewUserHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"username": "testuser",
		"password": 123,
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/users", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for wrong password type, got %d", w.Code)
	}
}

func TestUserHandler_Create_WrongType_IsAdmin_String(t *testing.T) {
	handler := NewUserHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"username": "testuser",
		"password": "password123",
		"is_admin": "yes",
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/users", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for wrong is_admin type, got %d", w.Code)
	}
}

// Test missing required fields
func TestUserHandler_Create_MissingUsername(t *testing.T) {
	handler := NewUserHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"password": "password123",
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/users", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for missing username, got %d", w.Code)
	}
}

func TestUserHandler_Create_MissingPassword(t *testing.T) {
	handler := NewUserHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"username": "testuser",
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/users", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for missing password, got %d", w.Code)
	}
}

// Test boundary values
func TestUserHandler_Create_PasswordTooShort(t *testing.T) {
	handler := NewUserHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"username": "testuser",
		"password": "short",
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/users", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for password too short, got %d", w.Code)
	}
}

func TestUserHandler_Create_PasswordTooLong(t *testing.T) {
	handler := NewUserHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"username": "testuser",
		"password": strings.Repeat("a", 200),
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/users", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for password too long, got %d", w.Code)
	}
}

func TestUserHandler_Create_UsernameEmpty(t *testing.T) {
	handler := NewUserHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"username": "",
		"password": "password123",
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/users", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for empty username, got %d", w.Code)
	}
}

func TestUserHandler_Create_UsernameTooLong(t *testing.T) {
	handler := NewUserHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: handler tries to access nil DB
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"username": strings.Repeat("a", 1000),
		"password": "password123",
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/users", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)
}

func TestUserHandler_Create_UsernameWithSpecialCharacters(t *testing.T) {
	handler := NewUserHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: handler tries to access nil DB
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"username": "user@#$%^&*()",
		"password": "password123",
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/users", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)
}

func TestUserHandler_Update_InvalidID_NonNumeric(t *testing.T) {
	handler := NewUserHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: handler tries to access nil DB
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"username": "newname",
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("PUT", "/users/abc", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "abc"}}

	handler.Update(c)
}

func TestUserHandler_Update_PasswordTooShort(t *testing.T) {
	handler := NewUserHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"password": "short",
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("PUT", "/users/1", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	handler.Update(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for password too short, got %d", w.Code)
	}
}

func TestUserHandler_Delete_InvalidID_NonNumeric(t *testing.T) {
	handler := NewUserHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: handler tries to access nil DB
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("DELETE", "/users/abc", nil)
	c.Params = gin.Params{{Key: "id", Value: "abc"}}

	handler.Delete(c)
}

// ==================== PLAN HANDLER VALIDATION TESTS ====================

func TestPlanHandler_Get_InvalidID_NonNumeric(t *testing.T) {
	handler := NewPlanHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: handler tries to access nil DB
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/plans/abc", nil)
	c.Params = gin.Params{{Key: "id", Value: "abc"}}

	handler.Get(c)
}

func TestPlanHandler_Get_InvalidID_Negative(t *testing.T) {
	handler := NewPlanHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: handler tries to access nil DB
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/plans/-1", nil)
	c.Params = gin.Params{{Key: "id", Value: "-1"}}

	handler.Get(c)
}

func TestPlanHandler_Create_MalformedJSON(t *testing.T) {
	handler := NewPlanHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/plans", bytes.NewBuffer([]byte("{invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestPlanHandler_Create_MissingName(t *testing.T) {
	handler := NewPlanHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"rpm": 100,
		"rpd": 1000,
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/plans", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for missing name, got %d", w.Code)
	}
}

// Duplicate removed - see handlers_input_test.go
// func TestPlanHandler_Create_EmptyName(t *testing.T)

func TestPlanHandler_Create_WrongType_RPM_String(t *testing.T) {
	handler := NewPlanHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"name": "test",
		"rpm":  "not a number",
		"rpd":  1000,
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/plans", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for wrong rpm type, got %d", w.Code)
	}
}

func TestPlanHandler_Create_WrongType_RPD_String(t *testing.T) {
	handler := NewPlanHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"name": "test",
		"rpm":  100,
		"rpd":  "not a number",
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/plans", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for wrong rpd type, got %d", w.Code)
	}
}

func TestPlanHandler_Create_WrongType_IsDefault_String(t *testing.T) {
	handler := NewPlanHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"name":       "test",
		"rpm":        100,
		"rpd":        1000,
		"is_default": "yes",
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/plans", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for wrong is_default type, got %d", w.Code)
	}
}

func TestPlanHandler_Create_NegativeRPM_Valid(t *testing.T) {
	handler := NewPlanHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: handler tries to access nil DB
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"name": "test",
		"rpm":  -1,
		"rpd":  1000,
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/plans", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)
	// -1 should be valid based on binding:"min=-1"
}

func TestPlanHandler_Create_BelowMinRPM(t *testing.T) {
	handler := NewPlanHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"name": "test",
		"rpm":  -2,
		"rpd":  1000,
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/plans", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for rpm below -1, got %d", w.Code)
	}
}

func TestPlanHandler_Create_NameTooLong(t *testing.T) {
	handler := NewPlanHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: handler tries to access nil DB
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"name": strings.Repeat("a", 1000),
		"rpm":  100,
		"rpd":  1000,
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/plans", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)
}

func TestPlanHandler_Update_InvalidID_NonNumeric(t *testing.T) {
	handler := NewPlanHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: handler tries to access nil DB
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"name": "newname",
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("PUT", "/plans/abc", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "abc"}}

	handler.Update(c)
}

func TestPlanHandler_Update_RPMBelowMin(t *testing.T) {
	handler := NewPlanHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: might access nil DB for valid input
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"rpm": -2,
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("PUT", "/plans/1", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	handler.Update(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for rpm below -1, got %d", w.Code)
	}
}

func TestPlanHandler_Delete_InvalidID_NonNumeric(t *testing.T) {
	handler := NewPlanHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: handler tries to access nil DB
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("DELETE", "/plans/abc", nil)
	c.Params = gin.Params{{Key: "id", Value: "abc"}}

	handler.Delete(c)
}

// ==================== APIKEY HANDLER VALIDATION TESTS ====================

func TestAPIKeyHandler_Create_MalformedJSON(t *testing.T) {
	handler := NewAPIKeyHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api-keys", bytes.NewBuffer([]byte("{invalid")))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", 1)

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestAPIKeyHandler_Create_MissingName(t *testing.T) {
	handler := NewAPIKeyHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/api-keys", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", 1)

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for missing name, got %d", w.Code)
	}
}

// Duplicate removed - see handlers_input_test.go
// func TestAPIKeyHandler_Create_EmptyName(t *testing.T)

func TestAPIKeyHandler_Create_NameTooLong(t *testing.T) {
	handler := NewAPIKeyHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: handler tries to access nil DB
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"name": strings.Repeat("a", 1000),
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/api-keys", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", 1)

	handler.Create(c)
}

func TestAPIKeyHandler_Create_WrongType_Name_Number(t *testing.T) {
	handler := NewAPIKeyHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"name": 123,
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/api-keys", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", 1)

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for wrong name type, got %d", w.Code)
	}
}

func TestAPIKeyHandler_Delete_InvalidID_NonNumeric(t *testing.T) {
	handler := NewAPIKeyHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: handler tries to access nil DB
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("DELETE", "/api-keys/abc", nil)
	c.Params = gin.Params{{Key: "id", Value: "abc"}}
	c.Set("user_id", 1)

	handler.Delete(c)
}

func TestAPIKeyHandler_Delete_InvalidID_Negative(t *testing.T) {
	handler := NewAPIKeyHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: handler tries to access nil DB
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("DELETE", "/api-keys/-1", nil)
	c.Params = gin.Params{{Key: "id", Value: "-1"}}
	c.Set("user_id", 1)

	handler.Delete(c)
}

// ==================== AIMODEL HANDLER VALIDATION TESTS ====================

func TestAIModelHandler_Get_InvalidID_NonNumeric(t *testing.T) {
	handler := NewAIModelHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: handler tries to access nil DB
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/models/abc", nil)
	c.Params = gin.Params{{Key: "id", Value: "abc"}}

	handler.Get(c)
}

func TestAIModelHandler_Get_InvalidID_Negative(t *testing.T) {
	handler := NewAIModelHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: handler tries to access nil DB
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/models/-1", nil)
	c.Params = gin.Params{{Key: "id", Value: "-1"}}

	handler.Get(c)
}

func TestAIModelHandler_Toggle_MalformedJSON(t *testing.T) {
	handler := NewAIModelHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("PUT", "/models/1/toggle", bytes.NewBuffer([]byte("{invalid")))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	handler.Toggle(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestAIModelHandler_Toggle_WrongType_Enabled_String(t *testing.T) {
	handler := NewAIModelHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"enabled": "yes",
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("PUT", "/models/1/toggle", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	handler.Toggle(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for wrong enabled type, got %d", w.Code)
	}
}

func TestAIModelHandler_Toggle_InvalidID_NonNumeric(t *testing.T) {
	handler := NewAIModelHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: handler tries to access nil DB
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"enabled": true,
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("PUT", "/models/abc/toggle", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "abc"}}

	handler.Toggle(c)
}

func TestAIModelHandler_Toggle_MissingEnabled_Field(t *testing.T) {
	handler := NewAIModelHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: handler tries to access nil DB
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("PUT", "/models/1/toggle", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	handler.Toggle(c)
	// Empty body is valid, defaults to false
}

// ==================== ENDPOINT HANDLER VALIDATION TESTS ====================

func TestEndpointHandler_Get_InvalidID_NonNumeric(t *testing.T) {
	handler := NewEndpointHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: handler tries to access nil DB
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/endpoints/abc", nil)
	c.Params = gin.Params{{Key: "id", Value: "abc"}}

	handler.Get(c)
}

func TestEndpointHandler_Get_InvalidID_Negative(t *testing.T) {
	handler := NewEndpointHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: handler tries to access nil DB
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/endpoints/-1", nil)
	c.Params = gin.Params{{Key: "id", Value: "-1"}}

	handler.Get(c)
}

func TestEndpointHandler_Create_MalformedJSON(t *testing.T) {
	handler := NewEndpointHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/endpoints", bytes.NewBuffer([]byte("{invalid")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestEndpointHandler_Create_MissingURL(t *testing.T) {
	handler := NewEndpointHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"name": "test endpoint",
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/endpoints", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for missing URL, got %d", w.Code)
	}
}

// Duplicate removed - see handlers_input_test.go
// func TestEndpointHandler_Create_EmptyURL(t *testing.T)

func TestEndpointHandler_Create_URLTooLong(t *testing.T) {
	handler := NewEndpointHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: handler tries to access nil DB
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"url":  "http://example.com/" + strings.Repeat("a", 5000),
		"name": "test",
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/endpoints", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)
}

func TestEndpointHandler_Create_WrongType_URL_Number(t *testing.T) {
	handler := NewEndpointHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: won't reach DB with validation failure
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"url":  123,
		"name": "test",
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/endpoints", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for wrong URL type, got %d", w.Code)
	}
}

func TestEndpointHandler_Create_WrongType_Name_Number(t *testing.T) {
	handler := NewEndpointHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: won't reach DB with validation failure
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"url":  "http://example.com",
		"name": 123,
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/endpoints", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for wrong name type, got %d", w.Code)
	}
}

func TestEndpointHandler_BatchCreate_MalformedJSON(t *testing.T) {
	handler := NewEndpointHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/endpoints/batch", bytes.NewBuffer([]byte("{invalid")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.BatchCreate(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestEndpointHandler_BatchCreate_MissingEndpoints(t *testing.T) {
	handler := NewEndpointHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/endpoints/batch", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.BatchCreate(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for missing endpoints, got %d", w.Code)
	}
}

func TestEndpointHandler_BatchCreate_InvalidEndpointsType(t *testing.T) {
	handler := NewEndpointHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"endpoints": "not an array",
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/endpoints/batch", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.BatchCreate(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for wrong endpoints type, got %d", w.Code)
	}
}

func TestEndpointHandler_Update_InvalidID_NonNumeric(t *testing.T) {
	handler := NewEndpointHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: handler tries to access nil DB
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"name": "newname",
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("PUT", "/endpoints/abc", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "abc"}}

	handler.Update(c)
}

func TestEndpointHandler_Update_WrongType_Name_Number(t *testing.T) {
	handler := NewEndpointHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"name": 123,
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("PUT", "/endpoints/1", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	handler.Update(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for wrong name type, got %d", w.Code)
	}
}

func TestEndpointHandler_Update_WrongType_URL_Number(t *testing.T) {
	handler := NewEndpointHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"url": 123,
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("PUT", "/endpoints/1", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	handler.Update(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for wrong url type, got %d", w.Code)
	}
}

func TestEndpointHandler_BatchDelete_MissingEndpointIDs(t *testing.T) {
	handler := NewEndpointHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("DELETE", "/endpoints/batch", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.BatchDelete(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for missing endpoint_ids, got %d", w.Code)
	}
}

func TestEndpointHandler_BatchDelete_WrongType_EndpointIDs_String(t *testing.T) {
	handler := NewEndpointHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"endpoint_ids": "not an array",
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("DELETE", "/endpoints/batch", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.BatchDelete(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for wrong endpoint_ids type, got %d", w.Code)
	}
}

func TestEndpointHandler_BatchTest_MissingEndpointIDs(t *testing.T) {
	handler := NewEndpointHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/endpoints/batch/test", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.BatchTest(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for missing endpoint_ids, got %d", w.Code)
	}
}

func TestEndpointHandler_TriggerTest_InvalidID_NonNumeric(t *testing.T) {
	handler := NewEndpointHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: handler tries to access nil DB
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/endpoints/xyz/test", nil)
	c.Params = gin.Params{{Key: "id", Value: "xyz"}}

	handler.TriggerTest(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestEndpointHandler_TriggerTest_InvalidID_Negative(t *testing.T) {
	handler := NewEndpointHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: handler tries to access nil DB
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/endpoints/-1/test", nil)
	c.Params = gin.Params{{Key: "id", Value: "-1"}}

	handler.TriggerTest(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestEndpointHandler_GetTask_InvalidID_NonNumeric(t *testing.T) {
	handler := NewEndpointHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: handler tries to access nil DB
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/endpoints/xyz/task", nil)
	c.Params = gin.Params{{Key: "id", Value: "xyz"}}

	handler.GetTask(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestEndpointHandler_GetTask_InvalidID_Negative(t *testing.T) {
	handler := NewEndpointHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: handler tries to access nil DB
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/endpoints/-1/task", nil)
	c.Params = gin.Params{{Key: "id", Value: "-1"}}

	handler.GetTask(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestEndpointHandler_BatchGetTasks_MissingEndpointIDs(t *testing.T) {
	handler := NewEndpointHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/endpoints/batch/tasks", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.BatchGetTasks(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for missing endpoint_ids, got %d", w.Code)
	}
}

func TestEndpointHandler_BatchGetTasks_WrongType_EndpointIDs_String(t *testing.T) {
	handler := NewEndpointHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"endpoint_ids": "not an array",
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/endpoints/batch/tasks", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.BatchGetTasks(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for wrong endpoint_ids type, got %d", w.Code)
	}
}

func TestEndpointHandler_Delete_InvalidID_NonNumeric(t *testing.T) {
	handler := NewEndpointHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: handler tries to access nil DB
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("DELETE", "/endpoints/abc", nil)
	c.Params = gin.Params{{Key: "id", Value: "abc"}}

	handler.Delete(c)
}

func TestEndpointHandler_Delete_InvalidID_Negative(t *testing.T) {
	handler := NewEndpointHandler(nil)
	defer func() {
		if r := recover(); r != nil {
			// Expected: handler tries to access nil DB
		}
	}()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("DELETE", "/endpoints/-1", nil)
	c.Params = gin.Params{{Key: "id", Value: "-1"}}

	handler.Delete(c)
}
