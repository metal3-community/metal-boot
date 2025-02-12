package main

import (
	"log"
	"net/http"

	"github.com/appkins-org/go-redfish-uefi/api/redfish"
	"github.com/gin-gonic/gin"
)

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=config.yaml https://opendev.org/airship/go-redfish/raw/branch/master/spec/openapi.yaml

func main() {
	server := redfish.NewRedfishServer()

	err := server.SetupConfigFile()

	if err != nil {
		log.Default().Fatal(err)
	}

	h := gin.Default()

	redfish.RegisterHandlers(r, server)

	s := &http.Server{
		Handler: h,
		Addr:    "0.0.0.0:8080",
	}

	// And we serve HTTP until the world ends.
	log.Fatal(s.ListenAndServe())
}
