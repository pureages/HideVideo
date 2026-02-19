package handlers

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"
	"hidevideo/backend/config"
	"hidevideo/backend/database"
	"hidevideo/backend/models"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// LoginAttempt 登录尝试记录
type LoginAttempt struct {
	Count     int
	FirstFail time.Time
	LastFail  time.Time
}

// CaptchaInfo 验证码信息
type CaptchaInfo struct {
	Code   string
	Expire time.Time
}

var (
	loginAttempts = make(map[string]*LoginAttempt)
	loginMu       sync.RWMutex
	captchaMap    = make(map[string]*CaptchaInfo)
	captchaMu     sync.RWMutex
)

// Login 登录
func Login(c *gin.Context) {
	var req struct {
		Username   string `json:"username" binding:"required"`
		Password   string `json:"password" binding:"required"`
		Captcha    string `json:"captcha"`
		RememberMe bool   `json:"remember_me"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入用户名和密码"})
		return
	}

	// 验证验证码
	if req.Captcha != "" {
		captchaMu.RLock()
		captchaInfo, exists := captchaMap[c.ClientIP()]
		captchaMu.RUnlock()

		if !exists || captchaInfo == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "验证码已失效，请刷新"})
			return
		}

		if time.Since(captchaInfo.Expire) > 5*time.Minute {
			captchaMu.Lock()
			delete(captchaMap, c.ClientIP())
			captchaMu.Unlock()
			c.JSON(http.StatusBadRequest, gin.H{"error": "验证码已过期，请刷新"})
			return
		}

		if captchaInfo.Code != req.Captcha {
			c.JSON(http.StatusBadRequest, gin.H{"error": "验证码错误"})
			return
		}

		// 验证成功后删除验证码
		captchaMu.Lock()
		delete(captchaMap, c.ClientIP())
		captchaMu.Unlock()
	}

	// 获取客户端IP
	clientIP := c.ClientIP()

	// 检查登录保护是否开启
	if config.GetLoginProtectionEnabled() {
		// 检查IP是否被锁定
		loginMu.RLock()
		attempt, exists := loginAttempts[clientIP]
		loginMu.RUnlock()

		if exists && attempt.Count >= config.GetMaxAttempts() {
			// 检查是否超过锁定时间
			if time.Since(attempt.LastFail) < config.GetLockoutTime() {
				remaining := config.GetLockoutTime() - time.Since(attempt.LastFail)
				minutes := int(remaining.Minutes())
				seconds := int(remaining.Seconds()) % 60
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error":           "登录尝试次数过多，请稍后再试",
					"lockout_seconds": remaining.Seconds(),
					"lockout_text":    fmt.Sprintf("%d分%d秒后重试", minutes, seconds),
				})
				return
			} else {
				// 锁定时间已过，清除记录
				loginMu.Lock()
				delete(loginAttempts, clientIP)
				loginMu.Unlock()
			}
		}
	}

	var user models.User
	result := database.DB.Where("username = ?", req.Username).First(&user)
	if result.Error != nil {
		// 记录登录失败
		if config.GetLoginProtectionEnabled() {
			recordLoginFailure(clientIP)
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		// 记录登录失败
		if config.GetLoginProtectionEnabled() {
			recordLoginFailure(clientIP)

			// 获取当前失败次数
			loginMu.RLock()
			currentAttempt := loginAttempts[clientIP]
			loginMu.RUnlock()

			if currentAttempt != nil {
				remaining := config.GetMaxAttempts() - currentAttempt.Count
				c.JSON(http.StatusUnauthorized, gin.H{
					"error":              "用户名或密码错误",
					"failed_attempts":    currentAttempt.Count,
					"remaining_attempts": remaining,
				})
				return
			}
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// 设置session
	session := sessions.Default(c)
	session.Set("user_id", user.ID)
	session.Set("username", user.Username)
	session.Set("role", user.Role)

	// 根据remember_me设置session过期时间
	maxAge := 24 * 3600 // 默认1天
	if req.RememberMe {
		maxAge = 365 * 24 * 3600 // 长期登录：1年
	}
	session.Options(sessions.Options{
		Path:     "/",
		MaxAge:   maxAge,
		SameSite: http.SameSiteLaxMode,
	})

	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Session保存失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "登录成功",
		"username": user.Username,
		"role":     user.Role,
	})
}

// Logout 登出
func Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "登出失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "登出成功"})
}

// recordLoginFailure 记录登录失败
func recordLoginFailure(ip string) {
	loginMu.Lock()
	defer loginMu.Unlock()

	attempt, exists := loginAttempts[ip]
	if !exists {
		loginAttempts[ip] = &LoginAttempt{
			Count:     1,
			FirstFail: time.Now(),
			LastFail:  time.Now(),
		}
	} else {
		// 如果距离上次失败超过30分钟，重置计数
		if time.Since(attempt.LastFail) > 30*time.Minute {
			attempt.Count = 1
			attempt.FirstFail = time.Now()
		} else {
			attempt.Count++
		}
		attempt.LastFail = time.Now()
	}
}

// CheckAuth 检查登录状态
func CheckAuth(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id")
	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"is_login": false})
		return
	}

	username := session.Get("username")
	role := session.Get("role")
	if role == nil {
		role = "member"
	}
	c.JSON(http.StatusOK, gin.H{
		"is_login": true,
		"username": username,
		"role":     role,
	})
}

// GetCaptcha 获取验证码
func GetCaptcha(c *gin.Context) {
	clientIP := c.ClientIP()

	// 生成随机4位数字验证码
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	code := fmt.Sprintf("%04d", r.Intn(10000))

	// 存储验证码
	captchaMu.Lock()
	captchaMap[clientIP] = &CaptchaInfo{
		Code:   code,
		Expire: time.Now().Add(5 * time.Minute),
	}
	captchaMu.Unlock()

	// 生成简单的SVG验证码图片
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="100" height="36" viewBox="0 0 100 36">
		<rect width="100" height="36" fill="#f0f0f0"/>
		<text x="10" y="25" font-family="Arial" font-size="20" fill="#333" font-weight="bold">%s</text>
		<line x1="0" y1="10" x2="100" y2="10" stroke="#ccc" stroke-width="1"/>
		<line x1="0" y1="26" x2="100" y2="26" stroke="#ccc" stroke-width="1"/>
	</svg>`, code)

	c.JSON(http.StatusOK, gin.H{
		"svg": svg,
	})
}

// AuthMiddleware 登录中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		userID := session.Get("user_id")
		if userID == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
			c.Abort()
			return
		}
		c.Set("user_id", userID)
		c.Next()
	}
}
