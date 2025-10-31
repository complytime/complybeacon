package server

import (
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/complytime/complybeacon/compass/api"
	compass "github.com/complytime/complybeacon/compass/service"
)

func NewGinServer(service *compass.Service, port string) *http.Server {
	r := gin.Default()

	api.RegisterHandlers(r, service)

	s := &http.Server{
		Handler:           r,
		Addr:              net.JoinHostPort("0.0.0.0", port),
		ReadHeaderTimeout: 10 * time.Second,
	}

	return s
}

func SetupTLS(server *http.Server, config Config) (string, string) {
	// TODO: Allow loosening here through configuration
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS13}
	server.TLSConfig = tlsConfig

	if config.Certificate.PublicKey == "" {
		log.Fatal("Invalid certification configuration. Please add certConfig.cert to the configuration.")
	}

	if config.Certificate.PrivateKey == "" {
		log.Fatal("Invalid certification configuration. Please add certConfig.key to the configuration.")
	}

	return config.Certificate.PublicKey, config.Certificate.PrivateKey
}
