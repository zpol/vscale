package api

import (
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/zpol/vscale/internal/allocator"
	"github.com/zpol/vscale/internal/auth"
	"github.com/zpol/vscale/internal/config"
	"github.com/zpol/vscale/internal/models"
	"gorm.io/gorm"
)

type Server struct {
	DB     *gorm.DB
	Auth   *auth.Service
	Alloc  *allocator.Allocator
	Cfg    config.Config
	Router *gin.Engine
}

func NewServer(db *gorm.DB, authSvc *auth.Service, alloc *allocator.Allocator, cfg config.Config) *Server {
	s := &Server{DB: db, Auth: authSvc, Alloc: alloc, Cfg: cfg}
	r := gin.Default()
	corsCfg := cors.DefaultConfig()
	if cfg.CORSOrigins == "*" {
		corsCfg.AllowAllOrigins = true
	} else {
		corsCfg.AllowOrigins = strings.Split(cfg.CORSOrigins, ",")
	}
	corsCfg.AllowHeaders = append(corsCfg.AllowHeaders, "Authorization", "X-Vscale-Token", "Content-Type")
	r.Use(cors.New(corsCfg))

	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

	api := r.Group("/api")
	{
		api.POST("/auth/login", s.login)
		api.POST("/auth/register", s.register)

		authed := api.Group("")
		authed.Use(authSvc.Middleware())
		{
			authed.GET("/auth/me", s.me)
			authed.GET("/stats", s.stats)
			authed.GET("/stats/history", s.statsHistory)

			authed.GET("/users", auth.RequireRoles(models.RoleAdmin, models.RoleOperator), s.listUsers)
			authed.POST("/users", auth.RequireRoles(models.RoleAdmin), s.createUser)
			authed.PUT("/users/:id", auth.RequireRoles(models.RoleAdmin), s.updateUser)
			authed.DELETE("/users/:id", auth.RequireRoles(models.RoleAdmin), s.deleteUser)

			authed.GET("/vscales", s.listVscales)
			authed.POST("/vscales", auth.RequireRoles(models.RoleAdmin, models.RoleOperator), s.createVscale)
			authed.GET("/vscales/:id", s.getVscale)
			authed.PUT("/vscales/:id", s.updateVscale)
			authed.DELETE("/vscales/:id", auth.RequireRoles(models.RoleAdmin, models.RoleOperator), s.deleteVscale)

			authed.GET("/vscales/:id/hubs", s.listHubs)
			authed.POST("/vscales/:id/hubs", s.createHub)
			authed.DELETE("/vscales/:id/hubs/:hubId", s.deleteHub)
			authed.POST("/vscales/:id/hubs/:hubId/token", s.rotateHubToken)

			authed.GET("/vscales/:id/devices", s.listDevices)
			authed.DELETE("/vscales/:id/devices/:deviceId", s.revokeDevice)
			authed.PUT("/vscales/:id/devices/:deviceId", s.updateDevice)

			authed.GET("/vscales/:id/join-tokens", s.listJoinTokens)
			authed.POST("/vscales/:id/join-tokens", s.createJoinToken)
			authed.DELETE("/vscales/:id/join-tokens/:tokenId", s.revokeJoinToken)

			authed.GET("/vscales/:id/routes", s.listRoutes)
			authed.POST("/vscales/:id/routes", s.createRoute)
			authed.POST("/vscales/:id/routes/:routeId/approve", s.approveRoute)
			authed.POST("/vscales/:id/routes/:routeId/deny", s.denyRoute)
			authed.DELETE("/vscales/:id/routes/:routeId", s.deleteRoute)

			authed.GET("/vscales/:id/topology", s.getTopology)
			authed.GET("/vscales/:id/members", s.listMembers)
			authed.POST("/vscales/:id/members", s.addMember)
			authed.DELETE("/vscales/:id/members/:userId", s.removeMember)

			authed.POST("/allocator/check", s.checkCIDR)
		}

		agent := api.Group("/agent")
		{
			agent.POST("/enroll", s.agentEnroll)
			agentAuthed := agent.Group("")
			agentAuthed.Use(auth.AgentAuth(db))
			{
				agentAuthed.GET("/config", s.agentConfig)
				agentAuthed.POST("/heartbeat", s.agentHeartbeat)
				agentAuthed.GET("/peers", s.hubPeers) // hub mode
			}
		}
	}

	s.Router = r
	return s
}

func (s *Server) Run() error {
	return s.Router.Run(s.Cfg.HTTPAddr)
}
