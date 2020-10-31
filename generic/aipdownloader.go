package generic


import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

type CountryAis interface {
	LoadJsonFile(path string)
	initClient() http.Client
	Process()
}

type ConfigurationDataStruct struct {
	MainLocalDir string
	MergeDir     string
}

var ConfData ConfigurationDataStruct

func (cds *ConfigurationDataStruct) LoadConfigurationFile(path string) {
	// Open our jsonFile
	jsonFile, err := os.Open(path)
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}

	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)

	err = json.Unmarshal(byteValue, cds)
	if err != nil {
		fmt.Println("error:", err)
	}
}
