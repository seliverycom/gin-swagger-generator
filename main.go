package main

import (
	"embed"
	"fmt"
	"github.com/seliverycom/gin-swagger-generator/config"
	"github.com/seliverycom/gin-swagger-generator/generator"
	"os"
	"path/filepath"
	"strings"
)

//go:embed static/*
var staticFolder embed.FS

func main() {
	conf := config.Config{
		ApiPath:           "internal/api",
		GeneratedFileName: "server.go",
		SwaggerPath:       "swagger",
	}

	for _, token := range os.Args {
		arr := strings.SplitN(token, "=", 2)

		if arr[0] == "--source-path" || arr[0] == "-p" {
			conf.ApiPath = arr[1]
			continue
		}

		if arr[0] == "--dest-path" || arr[0] == "-d" {
			conf.ApiPath = arr[1]
			continue
		}

		if arr[0] == "--gen-name" || arr[0] == "-f" {
			cleared := filepath.Base(arr[1])
			if cleared != arr[1] {
				panic("Looks like you try to change folder. Param is for file name")
			}
			conf.GeneratedFileName = arr[1]
			continue
		}

		if arr[0] == "--swagger" || arr[0] == "-w" {
			conf.SwaggerPath = arr[1]
			continue
		}
	}

	fmt.Println("Starting generation...")
	fmt.Println("Path: ", conf.ApiPath)
	gen := generator.New(conf, staticFolder)
	gen.Init()
}
