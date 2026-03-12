package httpserver

import (
	"fmt"

	"github.com/smap-hcmut/shared-libs/go/response"

	"github.com/gin-gonic/gin"
)

const (
	healthMessage = "From Smap API V1 With Love"
	healthVersion = "1.0.0"
	serviceName   = "ingest-srv"
)

func (srv HTTPServer) healthCheck(c *gin.Context) {
	response.OK(c, gin.H{
		"status":  "healthy",
		"message": healthMessage,
		"version": healthVersion,
		"service": serviceName,
	})
}

func (srv HTTPServer) readyCheck(c *gin.Context) {
	ctx := c.Request.Context()

	postgresReady := false
	redisReady := false
	minioReady := false
	kafkaReady := false
	rabbitReady := false

	postgresErr := ""
	redisErr := ""
	minioErr := ""
	kafkaErr := ""
	rabbitErr := ""

	if srv.postgresDB != nil {
		if err := srv.postgresDB.PingContext(ctx); err == nil {
			postgresReady = true
		} else {
			postgresErr = err.Error()
		}
	} else {
		postgresErr = "not initialized"
	}

	if srv.redis != nil {
		if err := srv.redis.Ping(ctx); err == nil {
			redisReady = true
		} else {
			redisErr = err.Error()
		}
	} else {
		redisErr = "not initialized"
	}

	if srv.minio != nil {
		if err := srv.minio.HealthCheck(ctx); err == nil {
			minioReady = true
		} else {
			minioErr = err.Error()
		}
	} else {
		minioErr = "not initialized"
	}

	if srv.kafka != nil {
		if err := srv.kafka.HealthCheck(); err == nil {
			kafkaReady = true
		} else {
			kafkaErr = err.Error()
		}
	} else {
		kafkaErr = "not initialized"
	}

	if srv.rabbitmq != nil {
		if srv.rabbitmq.IsReady() {
			rabbitReady = true
		} else {
			rabbitErr = "connection is not ready"
		}
	} else {
		rabbitErr = "not initialized"
	}

	deps := gin.H{
		"postgres": gin.H{"ready": postgresReady, "error": postgresErr},
		"redis":    gin.H{"ready": redisReady, "error": redisErr},
		"minio":    gin.H{"ready": minioReady, "error": minioErr},
		"kafka":    gin.H{"ready": kafkaReady, "error": kafkaErr},
		"rabbitmq": gin.H{"ready": rabbitReady, "error": rabbitErr},
	}

	if !postgresReady || !redisReady {
		c.JSON(503, gin.H{
			"status":       "not ready",
			"message":      "Core dependencies not ready",
			"service":      serviceName,
			"dependencies": deps,
		})
		return
	}

	response.OK(c, gin.H{
		"status":       "ready",
		"message":      healthMessage,
		"version":      healthVersion,
		"service":      serviceName,
		"database":     "connected",
		"redis":        "connected",
		"dependencies": deps,
		"summary":      fmt.Sprintf("minio=%t kafka=%t rabbitmq=%t", minioReady, kafkaReady, rabbitReady),
	})
}

func (srv HTTPServer) liveCheck(c *gin.Context) {
	response.OK(c, gin.H{
		"status":  "alive",
		"message": healthMessage,
		"version": healthVersion,
		"service": serviceName,
	})
}
