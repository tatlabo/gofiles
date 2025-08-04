package main

import (
	"gofiles/internal/utils"
	"html/template"
	"io"
	"log"
	"net/http"
	"time"

	"gofiles/internal/handlers"

	_ "net/http/pprof"

	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var limit int

// JWT Secret key - In production, use environment variable
var jwtSecret = []byte("your-super-secret-jwt-key")

type Template struct {
	templates *template.Template
}

// Custom JWT Claims
type JWTCustomClaims struct {
	Name  string `json:"name"`
	Admin bool   `json:"admin"`
	jwt.RegisteredClaims
}

// Login credentials
type LoginRequest struct {
	Username string `json:"username" form:"username"`
	Password string `json:"password" form:"password"`
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

// Login handler
func login(c echo.Context) error {
	var req LoginRequest

	// Handle both JSON and form data
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format",
		})
	}

	// Check credentials (replace with your actual authentication logic)
	if req.Username == "admin" && req.Password == "password" {
		// Create token with claims
		claims := &JWTCustomClaims{
			Name:  req.Username,
			Admin: true,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 72)), // 72 hours
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				Subject:   req.Username,
			},
		}

		// Create token with claims
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

		// Generate encoded token
		tokenString, err := token.SignedString(jwtSecret)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to generate token",
			})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"token": tokenString,
			"user":  req.Username,
			"admin": true,
		})
	}

	return c.JSON(http.StatusUnauthorized, map[string]string{
		"error": "Invalid credentials",
	})
}

// Protected endpoint example
func protected(c echo.Context) error {
	user := c.Get("user").(*jwt.Token)
	claims := user.Claims.(*JWTCustomClaims)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Welcome to protected area!",
		"user":    claims.Name,
		"admin":   claims.Admin,
	})
}

// Admin middleware
func isAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		user := c.Get("user").(*jwt.Token)
		claims := user.Claims.(*JWTCustomClaims)

		if !claims.Admin {
			return c.JSON(http.StatusForbidden, map[string]string{
				"error": "Admin access required",
			})
		}

		return next(c)
	}
}

func main() {

	// Initialize Echo framework
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.CORS())
	e.Use(middleware.Recover())

	// JWT Config
	jwtConfig := echojwt.Config{
		NewClaimsFunc: func(c echo.Context) jwt.Claims {
			return new(JWTCustomClaims)
		},
		SigningKey: jwtSecret,
	}

	e.Static("/static", "static")
	e.Static("/media", "media")

	e.Renderer = &Template{
		templates: template.Must(template.New("").Funcs(template.FuncMap{
			"formatDate": utils.FormatDate, // Register the custom function
			"not":        utils.Not,
			"equals":     utils.Equals,
			"notequals":  utils.Notequals,
		}).ParseGlob("public/views/*.html")),
	}

	// Public routes (no JWT required)

	e.POST("/login", login)
	e.GET("/", handlers.StartPage)
	e.GET("/search", handlers.SearchInDb)
	e.POST("/search", handlers.SearchInDb)
	e.GET("/detail/:id", handlers.DetailById)
	e.GET("/details/:id", handlers.DetailById)
	e.GET("/preview/:id", handlers.PreviewById)
	e.GET("/preview/image/:id", handlers.PreviewImage)
	e.GET("/json/search", handlers.ResponseJson)
	e.GET("/append", handlers.ResponseAppend)
	e.GET("/access", Accessible)

	// Protected routes group
	protected := e.Group("/admin")
	protected.Use(echojwt.WithConfig(jwtConfig))

	// Admin-only routes
	e.GET("/dirs", handlers.AddPath)
	e.POST("/dirs", handlers.AddPath)
	e.POST("/scan", handlers.ScanDirectory)
	e.DELETE("/delete/:id", handlers.DeleteDirectory)
	e.GET("/test", handlers.TestEndpoint)

	// protected.GET("/dirs", handlers.AddPath, isAdmin)
	// protected.POST("/dirs", handlers.AddPath, isAdmin)
	// protected.POST("/scan", handlers.ScanDirectory, isAdmin)
	// protected.DELETE("/delete/:id", handlers.DeleteDirectory, isAdmin)
	// protected.GET("/test", handlers.TestEndpoint, isAdmin)

	// JWT protected route example
	protected.GET("/profile", func(c echo.Context) error {
		token := c.Get("user").(*jwt.Token)
		claims := token.Claims.(*JWTCustomClaims)
		return c.JSON(http.StatusOK, map[string]interface{}{
			"name":    claims.Name,
			"admin":   claims.Admin,
			"message": "This is a protected route",
		})
	})

	// Alternative: Apply JWT to specific routes individually
	// e.GET("/dirs", handlers.AddPath, echojwt.WithConfig(jwtConfig), isAdmin)
	// e.POST("/dirs", handlers.AddPath, echojwt.WithConfig(jwtConfig), isAdmin)
	// e.POST("/scan", handlers.ScanDirectory, echojwt.WithConfig(jwtConfig), isAdmin)
	// e.DELETE("/delete/:id", handlers.DeleteDirectory, echojwt.WithConfig(jwtConfig), isAdmin)

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	e.Logger.Fatal(e.Start(":80"))

	// log.Fatal(http.ListenAndServe("localhost:8000", nil))

}
