/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"github.com/23technologies/23kectl/cmd"
	"github.com/23technologies/23kectl/pkg/logger"
)

func main() {
	logger.Init()
	cmd.Execute()
}
