package main

import (
	"fmt"

	"github.com/NagoDede/aiploader/generic"
	"github.com/NagoDede/aiploader/japan"
)

func main() {
	generic.ConfData = generic.ConfigurationDataStruct{}
	japan.JapanAis = japan.JpData{}
	fmt.Println("AIP Downloader is starting")
	generic.ConfData.LoadConfigurationFile("./aipdownloader.json")
	fmt.Printf("Data will be stored in %s \n", generic.ConfData.MainLocalDir)

	japan.JapanAis.LoadJsonFile("./japan.json")
	japan.JapanAis.Process()
	fmt.Println("AIP Downloader - End of process")
}
