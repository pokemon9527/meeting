package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"meeting-go/internal/models"
	jwtUtil "meeting-go/pkg/jwt"
	"meeting-go/pkg/response"
)

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=2,max=30"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func Register(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}

		var existingUser models.User
		result := db.Where("email = ? OR username = ?", req.Email, req.Username).First(&existingUser)
		if result.RowsAffected > 0 {
			response.BadRequest(c, "用户已存在")
			return
		}

		user := models.User{
			Username: req.Username,
			Email:    req.Email,
		}
		if err := user.SetPassword(req.Password); err != nil {
			response.InternalError(c, "密码加密失败")
			return
		}

		if err := db.Create(&user).Error; err != nil {
			response.InternalError(c, "创建用户失败")
			return
		}

		token, _ := jwtUtil.GenerateToken(user.ID.String(), user.Username, user.Avatar)
		refreshToken, _ := jwtUtil.GenerateRefreshToken(user.ID.String())

		c.JSON(http.StatusCreated, gin.H{
			"success": true,
			"data": gin.H{
				"user":          user,
				"token":         token,
				"refresh_token": refreshToken,
			},
		})
	}
}

func Login(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}

		var user models.User
		result := db.Where("email = ?", req.Email).First(&user)
		if result.Error != nil {
			response.Unauthorized(c, "邮箱或密码错误")
			return
		}

		if !user.CheckPassword(req.Password) {
			response.Unauthorized(c, "邮箱或密码错误")
			return
		}

		token, _ := jwtUtil.GenerateToken(user.ID.String(), user.Username, user.Avatar)
		refreshToken, _ := jwtUtil.GenerateRefreshToken(user.ID.String())

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"user":          user,
				"token":         token,
				"refresh_token": refreshToken,
			},
		})
	}
}

func RefreshToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RefreshRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}

		userID, err := jwtUtil.ValidateRefreshToken(req.RefreshToken)
		if err != nil {
			response.Unauthorized(c, "无效的刷新Token")
			return
		}

		token, _ := jwtUtil.GenerateToken(userID, "", "")
		newRefreshToken, _ := jwtUtil.GenerateRefreshToken(userID)

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"token":         token,
				"refresh_token": newRefreshToken,
			},
		})
	}
}

func GetCurrentUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")

		var user models.User
		if err := db.Where("id = ?", userID).First(&user).Error; err != nil {
			response.NotFound(c, "用户不存在")
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    gin.H{"user": user},
		})
	}
}

// Helper to parse UUID
func parseUUID(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
}
