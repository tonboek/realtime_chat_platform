package handlers

import (
	"fmt"
	"net/http"
	"realtime_chat_platform/internal/database"
	"realtime_chat_platform/internal/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// возвращает профиль текущего пользователя
func GetProfileHandler(c *gin.Context) {
	uid := c.GetInt("user_id")
	if uid == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, uid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	fmt.Printf("Loading profile for user %d: avatar=%s\n", uid, user.Avatar)

	c.JSON(http.StatusOK, gin.H{
		"id":          user.ID,
		"username":    user.Username,
		"nickname":    user.Nickname,
		"avatar":      user.Avatar,
		"bio":         user.Bio,
		"last_active": user.LastActive,
		"created_at":  user.CreatedAt,
	})
}

// обновляет информацию о профиле пользователя
func UpdateProfileHandler(c *gin.Context) {
	uid := c.GetInt("user_id")
	if uid == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var request struct {
		Nickname string `json:"nickname"`
		Avatar   string `json:"avatar"`
		Bio      string `json:"bio"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, uid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// обновление полей
	updates := make(map[string]interface{})
	if request.Nickname != "" {
		updates["nickname"] = request.Nickname
	}
	if request.Avatar != "" {
		updates["avatar"] = request.Avatar
	}
	if request.Bio != "" {
		updates["bio"] = request.Bio
	}

	if len(updates) > 0 {
		if err := database.DB.Model(&user).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile updated successfully",
		"profile": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"nickname": user.Nickname,
			"avatar":   user.Avatar,
			"bio":      user.Bio,
		},
	})
}

// позволяет пользователям изменять свой пароль
func ChangePasswordHandler(c *gin.Context) {
	uid := c.GetInt("user_id")
	if uid == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var request struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, uid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(request.CurrentPassword)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Current password is incorrect"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	if err := database.DB.Model(&user).Update("password", string(hashedPassword)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// возвращает публичный профиль конкретного пользователя
func GetUserProfileHandler(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username is required"})
		return
	}

	var user models.User
	if err := database.DB.Where("username = ?", username).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          user.ID,
		"username":    user.Username,
		"nickname":    user.Nickname,
		"avatar":      user.Avatar,
		"bio":         user.Bio,
		"last_active": user.LastActive,
		"created_at":  user.CreatedAt,
	})
}
