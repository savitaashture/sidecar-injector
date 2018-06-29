package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"context"

	log "github.com/Sirupsen/logrus"
	"github.com/go-chassis/sidecar-injector/webhook"
	"github.com/go-chassis/sidecar-injector/loger"
)

func main() {
	var parms webhook.WebHookParameters

	loger.Initialize()
	// get command line parameters
	flag.IntVar(&parms.Port, "port", 443, "Webhook server port.")
	flag.StringVar(&parms.CertFile, "tlsCertFile", "/etc/webhook/mesher/certs/cert.pem", "File containing the x509 Certificate for HTTPS.")
	flag.StringVar(&parms.KeyFile, "tlsKeyFile", "/etc/webhook/mesher/certs/key.pem", "File containing the x509 private key to --tlsCertFile.")
	flag.StringVar(&parms.SidecarConfigFile, "sidecarCfgFile", "/etc/webhook/mesher/config/sidecarconfig.yaml", "File containing the mutation configuration.")
	flag.DurationVar(&parms.HealthCheckInterval, "healthCheckInterval", 0, "Configure how frequently the health chek interval updated.")
	flag.StringVar(&parms.HealthCheckFile, "healthCheckFile", "", "File that should be periodically updated if health check is enabled.")
	flag.Parse()

	wh, err := webhook.NewWebhook(parms)
	if err != nil {
		log.Errorf("failed to create webhook injection", err)
	}

	stop := make(chan struct{})
	go wh.Run(stop, parms)

	signalC := make(chan os.Signal, 1)
	signal.Notify(signalC, syscall.SIGINT, syscall.SIGTERM)
	<-signalC

	log.Infof("Shutting down wenhook server gracefully")
	wh.Server.Shutdown(context.Background())
	close(stop)
}
