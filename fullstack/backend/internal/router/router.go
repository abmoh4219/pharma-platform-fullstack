package router

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"pharma-platform/internal/config"
	"pharma-platform/internal/handler"
	"pharma-platform/internal/logging"
	"pharma-platform/internal/middleware"
)

func New(cfg config.Config, db *sql.DB) *gin.Engine {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	binding.EnableDecoderDisallowUnknownFields = true

	r := gin.New()
	r.HandleMethodNotAllowed = true
	if err := r.SetTrustedProxies(nil); err != nil {
		logging.Warn("router", "failed to set trusted proxies", map[string]any{"error": err.Error()})
	}

	r.Use(
		gin.CustomRecovery(func(c *gin.Context, recovered any) {
			logging.Error("router", "panic recovered", map[string]any{"recovered": recovered})
			middleware.AbortWithError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "internal server error")
		}),
		middleware.RequestContext(),
		middleware.SecurityHeaders(cfg.CORSOrigins),
		middleware.NewIPRateLimiter(cfg.RateLimitRPM).Middleware(),
	)
	r.NoRoute(func(c *gin.Context) {
		middleware.AbortWithError(c, http.StatusNotFound, "NOT_FOUND", "route not found")
	})
	r.NoMethod(func(c *gin.Context) {
		middleware.AbortWithError(c, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	})

	apiHandler, err := handler.NewAPI(cfg, db)
	if err != nil {
		logging.Error("router", "failed to initialize API handlers", map[string]any{"error": err.Error()})
		panic(err)
	}
	authMiddleware := middleware.NewAuthMiddleware(db, cfg)

	api := r.Group("/api/v1")
	{
		api.GET("/health", apiHandler.Health)
		api.POST("/auth/login", apiHandler.Login)
	}

	protected := api.Group("")
	protected.Use(authMiddleware.RequireAuth(), middleware.DataScopeRequired())
	{
		protected.GET("/auth/me", apiHandler.Me)
		protected.POST("/auth/logout", apiHandler.Logout)
		protected.PUT("/auth/users/:id/permissions", middleware.RequireRoles("system_admin"), apiHandler.UpdateUserPermission)

		protected.GET("/dashboard/summary", apiHandler.DashboardSummary)

		recruitment := protected.Group("/recruitment")
		recruitment.Use(middleware.RequireRoles("recruitment_specialist", "system_admin"))
		{
			recruitment.GET("/positions", apiHandler.ListPositions)
			recruitment.POST("/positions", apiHandler.CreatePosition)

			recruitment.GET("/candidates", apiHandler.ListCandidates)
			recruitment.POST("/candidates", apiHandler.CreateCandidate)
			recruitment.PUT("/candidates/:id", apiHandler.UpdateCandidate)
			recruitment.POST("/candidates/import", apiHandler.ImportCandidates)
			recruitment.POST("/candidates/merge", apiHandler.MergeCandidates)
			recruitment.GET("/candidates/search", apiHandler.SmartSearchCandidates)
			recruitment.GET("/candidates/:id/match-score", apiHandler.CandidateMatchScore)
			recruitment.GET("/candidates/:id/recommendations", apiHandler.CandidateRecommendations)
		}

		compliance := protected.Group("/compliance")
		compliance.Use(middleware.RequireRoles("compliance_admin", "system_admin"))
		{
			compliance.GET("/qualifications", apiHandler.ListQualifications)
			compliance.POST("/qualifications", apiHandler.CreateQualification)
			compliance.PUT("/qualifications/:id", apiHandler.UpdateQualification)
			compliance.DELETE("/qualifications/:id", apiHandler.DeleteQualification)

			compliance.GET("/restrictions", apiHandler.ListRestrictions)
			compliance.POST("/restrictions", apiHandler.CreateRestriction)
			compliance.PUT("/restrictions/:id", apiHandler.UpdateRestriction)
			compliance.DELETE("/restrictions/:id", apiHandler.DeleteRestriction)
			compliance.POST("/restrictions/check", apiHandler.CheckRestriction)
		}

		cases := protected.Group("/cases")
		cases.Use(middleware.RequireRoles("business_specialist", "compliance_admin", "recruitment_specialist", "system_admin"))
		{
			cases.GET("", apiHandler.ListCases)
			cases.POST("", apiHandler.CreateCase)
			cases.PUT("/:id/assign", apiHandler.AssignCase)
			cases.PUT("/:id/status", apiHandler.UpdateCaseStatus)
			cases.GET("/:id/attachments", apiHandler.ListCaseAttachments)
			cases.GET("/:id/history", apiHandler.ListCaseHistory)
		}

		files := protected.Group("/files")
		{
			files.POST("/initiate", apiHandler.InitiateUpload)
			files.POST("/chunk", apiHandler.UploadChunk)
			files.POST("/complete", apiHandler.CompleteUpload)
			files.GET("/sessions/:id", apiHandler.GetUploadSession)
			files.GET("/:id/download", apiHandler.DownloadAttachment)
		}

		audit := protected.Group("/audit")
		audit.Use(middleware.RequireRoles("compliance_admin", "system_admin"))
		{
			audit.GET("/logs", apiHandler.ListAuditLogs)
			audit.GET("/logs/export", apiHandler.ExportAuditLogs)
		}
	}

	return r
}
