package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"bitbucket.org/waseka/waseka-xml-generator/parser"
	"bitbucket.org/waseka/waseka-xml-generator/urlchecker"
	"bitbucket.org/waseka/waseka-xml-generator/utils"
)

var start time.Time

func init() {
	start = time.Now()
}

func main() {
	fmt.Println("main execution started at time", time.Since(start))
	executionTypePtr := flag.String("type", "parse", "Run program to parse or test")
	flag.Parse()

	executionType, err := utils.VerifyExecutionType(*executionTypePtr)

	if err != nil {
		panic(err.Error())
	}

	if executionType == "parse" {
		xmlParser()
	} else if executionType == "test" {
		urlChecker()
	}

	fmt.Println("\nmain execution stopped at time", time.Since(start))
}

func urlChecker() {
	urlchecker.CheckURL()
}

func xmlParser() {
	// remove existent contents from feeds directory of golang app
	utils.RemoveExistentContents("feeds")

	// parse each property category based on "PropertyTableMap" of utils
	for category := range utils.PropertyTableMap {
		parser.ParseToXML(category)
	}

	// check "IS_EXPORTABLE" from .env file to determine to transfer feeds from golang app
	// to "EXPORT_PATH" directroy of .env file
	isExportable := os.Getenv("IS_EXPORTABLE")
	if isExportable == "true" {
		utils.TransferFeeds()
	}
}
