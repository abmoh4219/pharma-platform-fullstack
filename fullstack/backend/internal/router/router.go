package router

import (
	"database/sql"
	"log"

	"github.com/gin-gonic/gin"

	"pharma-platform/internal/config"
	"pharma-platform/internal/handler"
	"pharma-platform/internal/middleware"
)

func New(cfg config.Config, db *sql.DB) *gin.Engine {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	apiHandler, err := handler.NewAPI(cfg, db)
	if err != nil {
		log.Fatalf("failed to initialize API handlers: %v", err)
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
