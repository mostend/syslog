package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"gopkg.in/mcuadros/go-syslog.v2"
	"gopkg.in/mcuadros/go-syslog.v2/format"
)

type syslogField struct {
	CertPath string
	KeyPath  string
	Host     string
	Format   string
	Port     string
	PeerName string
}

var RFC = map[string]format.Format{
	"RFC3164": syslog.RFC3164,
	"RFC5424": syslog.RFC5424,
	"RFC6587": syslog.RFC6587,
}

func main() {
	var configFile string
	flag.StringVar(&configFile, "c", "", "Configuration file")
	flag.Parse()
	if configFile == "" {
		log.Println("Configuration file is required")
		return
	}
	by, err := os.ReadFile(configFile)
	if err != nil {
		log.Println(err)
		return
	}

	var syslogField syslogField
	err = json.Unmarshal(by, &syslogField)
	if err != nil {
		log.Println("read config file err:", err)
		return
	}

	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)

	server := syslog.NewServer()
	server.SetFormat(RFC["RFC3164"])
	server.SetHandler(handler)

	cer, err := tls.LoadX509KeyPair(syslogField.CertPath, syslogField.KeyPath)
	if err != nil {
		log.Println(err)
		return
	}
	config := &tls.Config{Certificates: []tls.Certificate{cer}}

	address := fmt.Sprintf("%s:%s", syslogField.Host, syslogField.Port)

	server.ListenTCPTLS(address, config)

	server.SetTlsPeerNameFunc(func(tlsConn *tls.Conn) (tlsPeer string, ok bool) {
		if len(tlsConn.ConnectionState().PeerCertificates) < 1 {
			return syslogField.PeerName, true
		}
		return tlsConn.ConnectionState().PeerCertificates[0].Subject.CommonName, true
	})
	server.Boot()

	fileName := fmt.Sprintf("syslog_%s.log", time.Now().Format("2006-01-02-15-04-05"))
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		log.Println(err)
		return
	}
	defer file.Close()
	go func(channel syslog.LogPartsChannel) {
		for logParts := range channel {
			file.WriteString("============================================================================================\r\n")
			log.Println("============================================================================================")
			for k, v := range logParts {
				str := fmt.Sprintf("%s %v:%v\r\n", time.Now().Format("2006-01-02 15:04:05"), k, v)
				file.WriteString(str)
				log.Println(k+":", v)
			}
			file.WriteString("============================================================================================\r\n")
			log.Println("============================================================================================")
		}
	}(channel)

	log.Println("syslog server start")

	str := fmt.Sprintf("%s %s\r\n", time.Now().Format("2006-01-02 15:04:05"), "syslog server start")
	file.WriteString(str)
	server.Wait()
}
