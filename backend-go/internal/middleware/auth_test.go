package middleware

import (
"net/http"
"net/http/httptest"
"testing"
"time"

"github.com/gin-gonic/gin"
"github.com/timlzh/ollama-hack/internal/config"
"github.com/timlzh/ollama-hack/internal/models"
"github.com/timlzh/ollama-hack/internal/services"
)

func init() {
gin.SetMode(gin.TestMode)
}

// Helper to create a real AuthService with test config
func createTestAuthService() *services.AuthService {
cfg := &config.Config{
App: config.AppConfig{
SecretKey:                "test_secret_key",
AccessTokenExpireMinutes: 30,
},
}
return services.NewAuthService(nil, cfg)
}

// Helper to create a valid token for testing
func createValidTestToken(userID int, username string, isAdmin bool) string {
authService := createTestAuthService()
user := &models.User{
ID:       userID,
Username: username,
IsAdmin:  isAdmin,
}
token, _ := authService.GenerateToken(user)
return token
}

// ============== AdminMiddleware Tests ==============

func TestAdminMiddleware_WithAdminUser(t *testing.T) {
w := httptest.NewRecorder()
c, _ := gin.CreateTestContext(w)
c.Set("is_admin", true)

handler := AdminMiddleware()
handler(c)

if c.IsAborted() {
t.Error("Expected request to not be aborted for admin user")
}
}

func TestAdminMiddleware_WithNonAdminUser(t *testing.T) {
w := httptest.NewRecorder()
c, _ := gin.CreateTestContext(w)
c.Set("is_admin", false)

handler := AdminMiddleware()
handler(c)

if !c.IsAborted() {
t.Error("Expected request to be aborted for non-admin user")
}
if w.Code != http.StatusForbidden {
t.Errorf("Expected status 403, got %d", w.Code)
}
}

func TestAdminMiddleware_WithMissingIsAdmin(t *testing.T) {
w := httptest.NewRecorder()
c, _ := gin.CreateTestContext(w)
// Not setting is_admin

handler := AdminMiddleware()
handler(c)

if !c.IsAborted() {
t.Error("Expected request to be aborted when is_admin is missing")
}
if w.Code != http.StatusForbidden {
t.Errorf("Expected status 403, got %d", w.Code)
}
}

func TestAdminMiddleware_Integration(t *testing.T) {
router := gin.New()
router.Use(func(c *gin.Context) {
// Simulate authenticated user
c.Set("is_admin", true)
c.Next()
})
router.Use(AdminMiddleware())
router.GET("/admin", func(c *gin.Context) {
c.JSON(http.StatusOK, gin.H{"message": "admin access"})
})

req, _ := http.NewRequest("GET", "/admin", nil)
w := httptest.NewRecorder()
router.ServeHTTP(w, req)

if w.Code != http.StatusOK {
t.Errorf("Expected status 200, got %d", w.Code)
}
}

func TestAdminMiddleware_NonAdminIntegration(t *testing.T) {
router := gin.New()
router.Use(func(c *gin.Context) {
c.Set("is_admin", false)
c.Next()
})
router.Use(AdminMiddleware())
router.GET("/admin", func(c *gin.Context) {
c.JSON(http.StatusOK, gin.H{"message": "admin access"})
})

req, _ := http.NewRequest("GET", "/admin", nil)
w := httptest.NewRecorder()
router.ServeHTTP(w, req)

if w.Code != http.StatusForbidden {
t.Errorf("Expected status 403, got %d", w.Code)
}
}

// ============== AuthMiddleware Tests ==============

func TestAuthMiddleware_NoHeader(t *testing.T) {
w := httptest.NewRecorder()
c, _ := gin.CreateTestContext(w)
c.Request, _ = http.NewRequest("GET", "/", nil)

handler := AuthMiddleware(nil)
handler(c)

if !c.IsAborted() {
t.Error("Expected request to be aborted without Authorization header")
}
if w.Code != http.StatusUnauthorized {
t.Errorf("Expected status 401, got %d", w.Code)
}
}

func TestAuthMiddleware_InvalidFormat(t *testing.T) {
w := httptest.NewRecorder()
c, _ := gin.CreateTestContext(w)
c.Request, _ = http.NewRequest("GET", "/", nil)
c.Request.Header.Set("Authorization", "InvalidFormat")

handler := AuthMiddleware(nil)
handler(c)

if !c.IsAborted() {
t.Error("Expected request to be aborted with invalid header format")
}
if w.Code != http.StatusUnauthorized {
t.Errorf("Expected status 401, got %d", w.Code)
}
}

func TestAuthMiddleware_EmptyToken(t *testing.T) {
w := httptest.NewRecorder()
c, _ := gin.CreateTestContext(w)
c.Request, _ = http.NewRequest("GET", "/", nil)
c.Request.Header.Set("Authorization", "Bearer ")

handler := AuthMiddleware(nil)
handler(c)

// Empty token should still be treated as invalid
if !c.IsAborted() {
t.Error("Expected request to be aborted with empty token")
}
}

// ============== NEW: AuthMiddleware with Valid JWT Token ==============

func TestAuthMiddleware_ValidBearerToken_Admin(t *testing.T) {
authService := createTestAuthService()
validToken := createValidTestToken(42, "admin_user", true)

w := httptest.NewRecorder()
c, _ := gin.CreateTestContext(w)
c.Request, _ = http.NewRequest("GET", "/", nil)
c.Request.Header.Set("Authorization", "Bearer "+validToken)

handler := AuthMiddleware(authService)
handler(c)

// Should not abort for valid token
if c.IsAborted() {
t.Error("Expected request to not be aborted for valid token")
}

// Verify user_id and is_admin are set correctly
userID, exists := c.Get("user_id")
if !exists {
t.Error("Expected user_id to be set in context")
}
if userID != 42 {
t.Errorf("Expected user_id 42, got %v", userID)
}

isAdmin, exists := c.Get("is_admin")
if !exists {
t.Error("Expected is_admin to be set in context")
}
if !isAdmin.(bool) {
t.Error("Expected is_admin to be true")
}
}

func TestAuthMiddleware_ValidBearerToken_RegularUser(t *testing.T) {
authService := createTestAuthService()
validToken := createValidTestToken(99, "regular_user", false)

w := httptest.NewRecorder()
c, _ := gin.CreateTestContext(w)
c.Request, _ = http.NewRequest("GET", "/", nil)
c.Request.Header.Set("Authorization", "Bearer "+validToken)

handler := AuthMiddleware(authService)
handler(c)

// Should not abort for valid token
if c.IsAborted() {
t.Error("Expected request to not be aborted for valid token")
}

// Verify user_id and is_admin are set correctly
userID, exists := c.Get("user_id")
if !exists {
t.Error("Expected user_id to be set in context")
}
if userID != 99 {
t.Errorf("Expected user_id 99, got %v", userID)
}

isAdmin, exists := c.Get("is_admin")
if !exists {
t.Error("Expected is_admin to be set in context")
}
if isAdmin.(bool) {
t.Error("Expected is_admin to be false")
}
}

// ============== NEW: AuthMiddleware with Expired JWT Token ==============

func TestAuthMiddleware_ExpiredBearerToken(t *testing.T) {
cfg := &config.Config{
App: config.AppConfig{
SecretKey:                "test_secret_key",
AccessTokenExpireMinutes: -1, // Negative = expired
},
}
authService := services.NewAuthService(nil, cfg)

user := &models.User{
ID:       1,
Username: "testuser",
IsAdmin:  false,
}

expiredToken, _ := authService.GenerateToken(user)
time.Sleep(100 * time.Millisecond) // Ensure it's expired

w := httptest.NewRecorder()
c, _ := gin.CreateTestContext(w)
c.Request, _ = http.NewRequest("GET", "/", nil)
c.Request.Header.Set("Authorization", "Bearer "+expiredToken)

handler := AuthMiddleware(authService)
handler(c)

if !c.IsAborted() {
t.Error("Expected request to be aborted for expired token")
}
if w.Code != http.StatusUnauthorized {
t.Errorf("Expected status 401, got %d", w.Code)
}
}

func TestAuthMiddleware_InvalidBearerToken(t *testing.T) {
authService := createTestAuthService()

w := httptest.NewRecorder()
c, _ := gin.CreateTestContext(w)
c.Request, _ = http.NewRequest("GET", "/", nil)
c.Request.Header.Set("Authorization", "Bearer invalid.token.string")

handler := AuthMiddleware(authService)
handler(c)

if !c.IsAborted() {
t.Error("Expected request to be aborted for invalid bearer token")
}
if w.Code != http.StatusUnauthorized {
t.Errorf("Expected status 401, got %d", w.Code)
}
}

// ============== NEW: Extract User Info from Context ==============

func TestAuthMiddleware_ContextExtraction_RegularUser(t *testing.T) {
authService := createTestAuthService()
validToken := createValidTestToken(555, "context_test_user", false)

w := httptest.NewRecorder()
c, _ := gin.CreateTestContext(w)
c.Request, _ = http.NewRequest("GET", "/", nil)
c.Request.Header.Set("Authorization", "Bearer "+validToken)

var capturedUserID int
var capturedIsAdmin bool
handler := AuthMiddleware(authService)
finalHandler := func(c *gin.Context) {
userID, _ := c.Get("user_id")
isAdmin, _ := c.Get("is_admin")
capturedUserID = userID.(int)
capturedIsAdmin = isAdmin.(bool)
c.JSON(http.StatusOK, gin.H{"success": true})
}

handler(c)
if !c.IsAborted() {
finalHandler(c)
}

if capturedUserID != 555 {
t.Errorf("Expected extracted user_id 555, got %d", capturedUserID)
}
if capturedIsAdmin {
t.Error("Expected extracted is_admin to be false")
}
}

func TestAuthMiddleware_ContextExtraction_AdminUser(t *testing.T) {
authService := createTestAuthService()
validToken := createValidTestToken(777, "admin_context_user", true)

w := httptest.NewRecorder()
c, _ := gin.CreateTestContext(w)
c.Request, _ = http.NewRequest("GET", "/", nil)
c.Request.Header.Set("Authorization", "Bearer "+validToken)

var capturedUserID int
var capturedIsAdmin bool
handler := AuthMiddleware(authService)
finalHandler := func(c *gin.Context) {
userID, _ := c.Get("user_id")
isAdmin, _ := c.Get("is_admin")
capturedUserID = userID.(int)
capturedIsAdmin = isAdmin.(bool)
c.JSON(http.StatusOK, gin.H{"success": true})
}

handler(c)
if !c.IsAborted() {
finalHandler(c)
}

if capturedUserID != 777 {
t.Errorf("Expected extracted user_id 777, got %d", capturedUserID)
}
if !capturedIsAdmin {
t.Error("Expected extracted is_admin to be true")
}
}

// ============== NEW: Full Middleware Chain Integration ==============

func TestAuthMiddleware_FullChain_BearerToken(t *testing.T) {
authService := createTestAuthService()
validToken := createValidTestToken(123, "chain_test_user", false)

router := gin.New()
router.Use(AuthMiddleware(authService))
router.GET("/protected", func(c *gin.Context) {
userID, _ := c.Get("user_id")
isAdmin, _ := c.Get("is_admin")
c.JSON(http.StatusOK, gin.H{
"user_id": userID,
"is_admin": isAdmin,
})
})

req, _ := http.NewRequest("GET", "/protected", nil)
req.Header.Set("Authorization", "Bearer "+validToken)
w := httptest.NewRecorder()
router.ServeHTTP(w, req)

if w.Code != http.StatusOK {
t.Errorf("Expected status 200, got %d", w.Code)
}
}

func TestAuthMiddleware_FullChain_WithAdminMiddleware(t *testing.T) {
authService := createTestAuthService()
validToken := createValidTestToken(999, "admin_chain_user", true)

router := gin.New()
router.Use(AuthMiddleware(authService))
router.GET("/admin/protected", AdminMiddleware(), func(c *gin.Context) {
c.JSON(http.StatusOK, gin.H{"message": "admin access"})
})

req, _ := http.NewRequest("GET", "/admin/protected", nil)
req.Header.Set("Authorization", "Bearer "+validToken)
w := httptest.NewRecorder()
router.ServeHTTP(w, req)

if w.Code != http.StatusOK {
t.Errorf("Expected status 200 for admin, got %d", w.Code)
}
}

func TestAuthMiddleware_FullChain_NonAdminAccessingAdminRoute(t *testing.T) {
authService := createTestAuthService()
validToken := createValidTestToken(111, "regular_user_no_admin", false)

router := gin.New()
router.Use(AuthMiddleware(authService))
router.GET("/admin/protected", AdminMiddleware(), func(c *gin.Context) {
c.JSON(http.StatusOK, gin.H{"message": "admin access"})
})

req, _ := http.NewRequest("GET", "/admin/protected", nil)
req.Header.Set("Authorization", "Bearer "+validToken)
w := httptest.NewRecorder()
router.ServeHTTP(w, req)

if w.Code != http.StatusForbidden {
t.Errorf("Expected status 403 for non-admin, got %d", w.Code)
}
}

func TestAuthMiddleware_FullChain_NoAuthHeader_RejectedBeforeAdminCheck(t *testing.T) {
router := gin.New()
router.Use(AuthMiddleware(nil))
router.GET("/admin/protected", AdminMiddleware(), func(c *gin.Context) {
c.JSON(http.StatusOK, gin.H{"message": "admin access"})
})

req, _ := http.NewRequest("GET", "/admin/protected", nil)
w := httptest.NewRecorder()
router.ServeHTTP(w, req)

if w.Code != http.StatusUnauthorized {
t.Errorf("Expected status 401 for missing auth, got %d", w.Code)
}
}

// ============== NEW: Case Insensitive Bearer Token ==============

func TestAuthMiddleware_BearerToken_CaseInsensitive(t *testing.T) {
authService := createTestAuthService()
validToken := createValidTestToken(321, "case_test_user", false)

testCases := []string{
"Bearer " + validToken,
"bearer " + validToken,
"BEARER " + validToken,
"BeArEr " + validToken,
}

for _, authHeader := range testCases {
w := httptest.NewRecorder()
c, _ := gin.CreateTestContext(w)
c.Request, _ = http.NewRequest("GET", "/", nil)
c.Request.Header.Set("Authorization", authHeader)

handler := AuthMiddleware(authService)
handler(c)

if c.IsAborted() {
t.Errorf("Expected request with header '%s' to not be aborted", authHeader)
}
}
}

// ============== NEW: Multiple Different Tokens ==============

func TestAuthMiddleware_MultipleDifferentTokens(t *testing.T) {
authService := createTestAuthService()

testCases := []struct {
name    string
userID  int
username string
isAdmin bool
}{
{"User1", 1, "user1", false},
{"User2", 2, "user2", false},
{"Admin1", 100, "admin1", true},
{"Admin2", 200, "admin2", true},
}

for _, tc := range testCases {
t.Run(tc.name, func(t *testing.T) {
token := createValidTestToken(tc.userID, tc.username, tc.isAdmin)

w := httptest.NewRecorder()
c, _ := gin.CreateTestContext(w)
c.Request, _ = http.NewRequest("GET", "/", nil)
c.Request.Header.Set("Authorization", "Bearer "+token)

handler := AuthMiddleware(authService)
handler(c)

if c.IsAborted() {
t.Errorf("Expected request to not be aborted for %s", tc.name)
}

userID, _ := c.Get("user_id")
isAdmin, _ := c.Get("is_admin")

if userID.(int) != tc.userID {
t.Errorf("Expected userID %d, got %d", tc.userID, userID.(int))
}
if isAdmin.(bool) != tc.isAdmin {
t.Errorf("Expected isAdmin %v, got %v", tc.isAdmin, isAdmin.(bool))
}
})
}
}

// ============== NEW: Malformed Token Variations ==============

func TestAuthMiddleware_MalformedTokenVariations(t *testing.T) {
authService := createTestAuthService()

testCases := []string{
"invalid.token",
"a.b.c.d",
"onlyonepart",
"random.garbage.here",
"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
}

for _, badToken := range testCases {
w := httptest.NewRecorder()
c, _ := gin.CreateTestContext(w)
c.Request, _ = http.NewRequest("GET", "/", nil)
c.Request.Header.Set("Authorization", "Bearer "+badToken)

handler := AuthMiddleware(authService)
handler(c)

if !c.IsAborted() {
t.Errorf("Expected request with token '%s' to be aborted", badToken)
}
}
}


// ============== NEW: Context Values Integrity ==============

func TestAuthMiddleware_ContextValuesNotLeaking(t *testing.T) {
authService := createTestAuthService()
token1 := createValidTestToken(111, "user111", false)
token2 := createValidTestToken(222, "user222", true)

w1 := httptest.NewRecorder()
c1, _ := gin.CreateTestContext(w1)
c1.Request, _ = http.NewRequest("GET", "/", nil)
c1.Request.Header.Set("Authorization", "Bearer "+token1)
handler := AuthMiddleware(authService)
handler(c1)

w2 := httptest.NewRecorder()
c2, _ := gin.CreateTestContext(w2)
c2.Request, _ = http.NewRequest("GET", "/", nil)
c2.Request.Header.Set("Authorization", "Bearer "+token2)
handler(c2)

userID1, _ := c1.Get("user_id")
isAdmin1, _ := c1.Get("is_admin")

userID2, _ := c2.Get("user_id")
isAdmin2, _ := c2.Get("is_admin")

if userID1.(int) != 111 || isAdmin1.(bool) {
t.Error("First request context corrupted")
}
if userID2.(int) != 222 || !isAdmin2.(bool) {
t.Error("Second request context corrupted")
}
}
