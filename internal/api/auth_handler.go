package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/service"
	"go.uber.org/zap"
)

type AuthHandler struct {
	service service.AuthService
	logger  *zap.SugaredLogger
}

func NewAuthHandler(service service.AuthService, logger *zap.SugaredLogger) *AuthHandler {
	return &AuthHandler{service: service, logger: logger}
}

type SignupUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

func (h *AuthHandler) Signup(c *gin.Context) {
	var req SignupUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrWrongJSON.Error()})
		return
	}

	user := domain.User{
		Email:    req.Email,
		Password: req.Password,
	}
	err := h.service.Register(c.Request.Context(), &user)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUserAlreadyExists):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case errors.Is(err, domain.ErrDatabase), errors.Is(err, domain.ErrHashPassword):
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "пользователь зарегистрирован"})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var user domain.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.service.Login(c.Request.Context(), &user)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrDatabase):
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	token, err := h.service.GenerateToken(user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrTokenGeneration})
	}

	c.SetCookie("jwt", token, int(domain.TokenExpiration.Seconds()), "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{"message": "вход успешен"})
}
