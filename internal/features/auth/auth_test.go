package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"app/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestAuth(t *testing.T) (*Handler, Service, Repository) {
	db, err := database.ConnectTestDB()
	assert.NoError(t, err)

	repo := NewRepository(db)
	svc := NewService(repo, "secret_key_for_test")
	handler := NewHandler(svc)
	return handler, svc, repo
}

// mockRepo is a fake repository for testing error paths
type mockRepo struct {
	createErr                error
	findByUsernameResult     *database.User
	findByUsernameErr        error
	findByEmailResult        *database.User
	findByEmailErr           error
	findByUsernameOrEmail    *database.User
	findByUsernameOrEmailErr error
	findByIDResult           *database.User
	findByIDErr              error
	updateErr                error
}

func (m *mockRepo) Create(user *database.User) error { return m.createErr }
func (m *mockRepo) FindByUsername(username string) (*database.User, error) {
	return m.findByUsernameResult, m.findByUsernameErr
}
func (m *mockRepo) FindByEmail(email string) (*database.User, error) {
	return m.findByEmailResult, m.findByEmailErr
}
func (m *mockRepo) FindByUsernameOrEmail(identifier string) (*database.User, error) {
	return m.findByUsernameOrEmail, m.findByUsernameOrEmailErr
}
func (m *mockRepo) FindByID(id uint) (*database.User, error) {
	return m.findByIDResult, m.findByIDErr
}
func (m *mockRepo) Update(user *database.User) error { return m.updateErr }

func TestIsValidEmail(t *testing.T) {
	assert.False(t, isValidEmail(""))
	assert.False(t, isValidEmail("invalid-email"))
	assert.False(t, isValidEmail("user@"))
	assert.False(t, isValidEmail("@domain.com"))
	assert.False(t, isValidEmail("user@domain"))
	assert.False(t, isValidEmail("\"a@b\"@domain.com"))
	assert.True(t, isValidEmail("user@example.com"))
	assert.True(t, isValidEmail("user.name@sub.example.co.id"))
}

func TestAuth_Register(t *testing.T) {
	handler, _, _ := setupTestAuth(t)

	r := gin.New()
	r.POST("/register", handler.Register)

	// 1. Valid Member Registration
	bodyValid := RegisterRequest{
		Username: "testuser",
		Email:    "testuser@test.local",
		Password: "password123",
		Role:     "member",
	}
	jsonBody, _ := json.Marshal(bodyValid)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var res map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	assert.Equal(t, "success", res["status"])
	data := res["data"].(map[string]interface{})
	assert.NotEmpty(t, data["token"])
	user := data["user"].(map[string]interface{})
	assert.Equal(t, "testuser", user["username"])
	assert.Equal(t, "member", user["role"])

	// 2. Duplicate Username
	wDupU := httptest.NewRecorder()
	reqDupU, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(jsonBody))
	r.ServeHTTP(wDupU, reqDupU)
	assert.Equal(t, http.StatusBadRequest, wDupU.Code)

	// 3. Duplicate Email
	dupEmailBody, _ := json.Marshal(RegisterRequest{
		Username: "anotheruser",
		Email:    "testuser@test.local",
		Password: "password123",
	})
	wDupE := httptest.NewRecorder()
	reqDupE, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(dupEmailBody))
	r.ServeHTTP(wDupE, reqDupE)
	assert.Equal(t, http.StatusBadRequest, wDupE.Code)

	// 4. Admin Role Registration
	adminBody, _ := json.Marshal(RegisterRequest{
		Username: "adminuser",
		Email:    "adminuser@test.local",
		Password: "password123",
		Role:     "admin",
	})
	wAdmin := httptest.NewRecorder()
	reqAdmin, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(adminBody))
	r.ServeHTTP(wAdmin, reqAdmin)
	assert.Equal(t, http.StatusCreated, wAdmin.Code)

	// 5. Invalid Registration cases
	invalidCases := []RegisterRequest{
		{Username: "", Email: "no@user.local", Password: "password123"},
		{Username: "bademail", Email: "bukan-email", Password: "password123"},
		{Username: "shortpass", Email: "short@pass.local", Password: "123"},
	}
	for _, tc := range invalidCases {
		b, _ := json.Marshal(tc)
		wInv := httptest.NewRecorder()
		reqInv, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(b))
		r.ServeHTTP(wInv, reqInv)
		assert.Equal(t, http.StatusBadRequest, wInv.Code)
	}

	// 6. Bad JSON binding
	wBadJSON := httptest.NewRecorder()
	reqBadJSON, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBufferString("{invalid json"))
	r.ServeHTTP(wBadJSON, reqBadJSON)
	assert.Equal(t, http.StatusBadRequest, wBadJSON.Code)
}

func TestAuth_Register_InternalError(t *testing.T) {
	// Use mock repo that returns unexpected error on Create
	mock := &mockRepo{
		createErr:            errors.New("db connection lost"),
		findByUsernameResult: nil,
		findByUsernameErr:    errors.New("not found"),
		findByEmailResult:    nil,
		findByEmailErr:       errors.New("not found"),
	}
	svc := NewService(mock, "secret")
	handler := NewHandler(svc)

	r := gin.New()
	r.POST("/register", handler.Register)

	body, _ := json.Marshal(RegisterRequest{
		Username: "erruser",
		Email:    "err@test.local",
		Password: "password123",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAuth_Login(t *testing.T) {
	handler, _, _ := setupTestAuth(t)

	r := gin.New()
	r.POST("/register", handler.Register)
	r.POST("/login", handler.Login)

	// Register account first
	regBody, _ := json.Marshal(RegisterRequest{
		Username: "loginuser",
		Email:    "loginuser@test.local",
		Password: "password123",
	})
	wReg := httptest.NewRecorder()
	reqReg, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(regBody))
	r.ServeHTTP(wReg, reqReg)
	assert.Equal(t, http.StatusCreated, wReg.Code)

	// 1. Login with username
	loginUserBody, _ := json.Marshal(LoginRequest{
		Username: "loginuser",
		Password: "password123",
	})
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(loginUserBody))
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// 2. Login with email
	loginEmailBody, _ := json.Marshal(LoginRequest{
		Username: "loginuser@test.local",
		Password: "password123",
	})
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(loginEmailBody))
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	// 3. Login wrong password
	wrongPassBody, _ := json.Marshal(LoginRequest{
		Username: "loginuser",
		Password: "wrongpassword",
	})
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(wrongPassBody))
	r.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusUnauthorized, w3.Code)

	// 4. Login user not found
	noUserBody, _ := json.Marshal(LoginRequest{
		Username: "nonexistent",
		Password: "password123",
	})
	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(noUserBody))
	r.ServeHTTP(w4, req4)
	assert.Equal(t, http.StatusUnauthorized, w4.Code)

	// 5. Login missing fields
	emptyBody, _ := json.Marshal(LoginRequest{
		Username: "loginuser",
		Password: "",
	})
	w5 := httptest.NewRecorder()
	req5, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(emptyBody))
	r.ServeHTTP(w5, req5)
	assert.Equal(t, http.StatusBadRequest, w5.Code)

	// 6. Bad JSON
	w6 := httptest.NewRecorder()
	req6, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBufferString("{bad"))
	r.ServeHTTP(w6, req6)
	assert.Equal(t, http.StatusBadRequest, w6.Code)
}

type mockAuthService struct {
	Service
	loginErr error
}

func (m *mockAuthService) Login(req *LoginRequest) (*AuthResponse, error) {
	return nil, m.loginErr
}

func TestAuth_Login_InternalError(t *testing.T) {
	mock := &mockRepo{
		findByUsernameOrEmail:    nil,
		findByUsernameOrEmailErr: errors.New("db crashed"),
	}
	svc := NewService(mock, "secret")
	handler := NewHandler(svc)

	r := gin.New()
	r.POST("/login", handler.Login)

	body, _ := json.Marshal(LoginRequest{
		Username: "crashuser",
		Password: "password123",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
	r.ServeHTTP(w, req)
	// DB error -> ErrInvalidCredentials -> 401
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Mock service returning unexpected 500 error
	mockSvc := &mockAuthService{loginErr: errors.New("unexpected fatal")}
	handler500 := NewHandler(mockSvc)
	r500 := gin.New()
	r500.POST("/login", handler500.Login)
	w500 := httptest.NewRecorder()
	req500, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
	r500.ServeHTTP(w500, req500)
	assert.Equal(t, http.StatusInternalServerError, w500.Code)
}

func TestAuth_IsValidEmail_EdgeCases(t *testing.T) {
	// edge cases for the parts check
	assert.False(t, isValidEmail("user@@domain.com"))
	assert.False(t, isValidEmail("@"))
}

func TestAuth_Service_Login_DirectCallErrInvalidInput(t *testing.T) {
	mock := &mockRepo{}
	svc := NewService(mock, "secret")
	// Call Login directly with empty username -> ErrInvalidInput
	_, err := svc.Login(&LoginRequest{Username: "", Password: "pw"})
	assert.ErrorIs(t, err, ErrInvalidInput)
	// Call Login directly with empty password -> ErrInvalidInput
	_, err = svc.Login(&LoginRequest{Username: "user", Password: ""})
	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestAuth_Logout(t *testing.T) {
	handler, _, _ := setupTestAuth(t)

	r := gin.New()
	r.POST("/logout", handler.Logout)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/logout", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuth_Profile(t *testing.T) {
	handler, svc, repo := setupTestAuth(t)

	// Register user
	authRes, err := svc.Register(&RegisterRequest{
		Username: "profileuser",
		Email:    "profileuser@test.local",
		Password: "password123",
	})
	assert.NoError(t, err)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", authRes.User.ID)
		c.Next()
	})
	r.GET("/profile", handler.GetProfile)
	r.PUT("/profile", handler.UpdateProfile)

	// 1. Get profile OK
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodGet, "/profile", nil)
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// 2. Get profile not found
	rNotFound := gin.New()
	rNotFound.Use(func(c *gin.Context) {
		c.Set("user_id", uint(999999))
		c.Next()
	})
	rNotFound.GET("/profile", handler.GetProfile)
	wNF := httptest.NewRecorder()
	reqNF, _ := http.NewRequest(http.MethodGet, "/profile", nil)
	rNotFound.ServeHTTP(wNF, reqNF)
	assert.Equal(t, http.StatusNotFound, wNF.Code)

	// 3. Update profile OK (email + password)
	updateBody, _ := json.Marshal(UpdateProfileRequest{
		Email:    "updated@test.local",
		Password: "newpassword123",
	})
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPut, "/profile", bytes.NewBuffer(updateBody))
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	// Verify new password works
	loginRes, err := svc.Login(&LoginRequest{
		Username: "profileuser",
		Password: "newpassword123",
	})
	assert.NoError(t, err)
	assert.NotNil(t, loginRes)

	// 4. Update profile invalid email
	badEmailBody, _ := json.Marshal(UpdateProfileRequest{
		Email: "bukan-email",
	})
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest(http.MethodPut, "/profile", bytes.NewBuffer(badEmailBody))
	r.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusBadRequest, w3.Code)

	// 5. Update profile duplicate email
	_ = repo.Create(&database.User{
		Username: "otheruser",
		Email:    "other@test.local",
		Password: "pass",
		Role:     "member",
	})
	dupBody, _ := json.Marshal(UpdateProfileRequest{
		Email: "other@test.local",
	})
	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest(http.MethodPut, "/profile", bytes.NewBuffer(dupBody))
	r.ServeHTTP(w4, req4)
	assert.Equal(t, http.StatusBadRequest, w4.Code)

	// 6. Update profile short password
	shortPassBody, _ := json.Marshal(UpdateProfileRequest{
		Password: "123",
	})
	w5 := httptest.NewRecorder()
	req5, _ := http.NewRequest(http.MethodPut, "/profile", bytes.NewBuffer(shortPassBody))
	r.ServeHTTP(w5, req5)
	assert.Equal(t, http.StatusBadRequest, w5.Code)

	// 7. Update profile bad JSON
	w6 := httptest.NewRecorder()
	req6, _ := http.NewRequest(http.MethodPut, "/profile", bytes.NewBufferString("{bad"))
	r.ServeHTTP(w6, req6)
	assert.Equal(t, http.StatusBadRequest, w6.Code)

	// 8. Update profile user not found
	rNotFound.PUT("/profile", handler.UpdateProfile)
	w7 := httptest.NewRecorder()
	req7, _ := http.NewRequest(http.MethodPut, "/profile", bytes.NewBuffer(updateBody))
	rNotFound.ServeHTTP(w7, req7)
	assert.Equal(t, http.StatusNotFound, w7.Code)
}

func TestAuth_UpdateProfile_InternalError(t *testing.T) {
	// Mock: findByID succeeds, but update fails
	user := &database.User{
		ID:       1,
		Username: "testuser",
		Email:    "test@test.local",
		Role:     "member",
		Password: "hashedpass",
	}
	mock := &mockRepo{
		findByIDResult:    user,
		findByIDErr:       nil,
		findByEmailResult: nil,
		findByEmailErr:    errors.New("not found"),
		updateErr:         errors.New("db write error"),
	}
	svc := NewService(mock, "secret")
	handler := NewHandler(svc)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Next()
	})
	r.PUT("/profile", handler.UpdateProfile)

	// Update with valid email that triggers update error
	body, _ := json.Marshal(UpdateProfileRequest{
		Email: "newemail@test.local",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/profile", bytes.NewBuffer(body))
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
