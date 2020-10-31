package japan

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"
)

var JapanAis JpData

type JpLoginFormData struct {
	FormName string `json:"formName"`
	Password string `json:"password"`
	UserID   string `json:"userID"`
}

type JpData struct {
	MainDataConfig
	LoginData        JpLoginFormData `json:"loginData"`
	LoginPage        string          `json:"loginPage"`
	AipIndexPageName string
}

type MainDataConfig struct {
	CountryDir       string `json:"countryDir"`
	MainAipPage      string
	MainAipActiveURL string
}

/*
Load the JSON file used for the access to the Japan AIP.
The required password can be provided by an environment variable or
directly set in the Json file.
When the environement variable is used, the password definition shall respect
the syntax "Env: ENV_VARIABLE_NAME". The function will then retrieve the content
of the environment variable ENV_VARIABLE_NAME.
If the environment variable does not exist or is empty, it generates a panic.
To define an empty password, just set Password = ""  in the Json file.
The same beahavior is extended to the User ID.

*/
func (jpd *JpData) LoadJsonFile(path string) {
	// Open our jsonFile
	jsonFile, err := os.Open(path)
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}

	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)

	err = json.Unmarshal(byteValue, jpd)
	if err != nil {
		fmt.Println("error:", err)
	}

	//The password may be provided by an environment variable
	if strings.HasPrefix(jpd.LoginData.Password, "Env:") {
		var s = strings.TrimPrefix(jpd.LoginData.Password, "Env:")
		s = strings.TrimSpace(s)
		jpd.LoginData.Password = os.Getenv(s)

		if jpd.LoginData.Password == "" {
			panic(fmt.Sprintf("Password Environment variable: %s  not defined\n", s))
		}
	}

	//The UserID may be provided by an environment variable
	if strings.HasPrefix(jpd.LoginData.UserID, "Env:") {
		var s = strings.TrimPrefix(jpd.LoginData.UserID, "Env:")
		s = strings.TrimSpace(s)
		jpd.LoginData.UserID = os.Getenv(s)

		if jpd.LoginData.UserID == "" {
			panic(fmt.Sprintf("User ID Environment variable: %s  not defined\n", s))
		}
	}
}

func (jpd *JpData) Process() {
	client := jpd.InitClient()

	//retrieve the  AIP document and the active one
	var aipDocsList AipDocs

	fmt.Println("Retrieve the AIP Documents")
	aipDocsList = getAipDocuments(&client)
	fmt.Println("Retrieve the Active Document")
	activeAipDoc := aipDocsList.getActiveAipDoc()
	activeAipDoc.NextEffectiveDate = aipDocsList.GetNextDate(*activeAipDoc)
	activeAipDoc.CountryCode = jpd.CountryDir
	activeAipDoc.ProcessDate = time.Now()
	fmt.Println("Active Document Effective Date:" + activeAipDoc.EffectiveDate.Format("02-Jan-2006") +
		" Publication Date: " + activeAipDoc.PublicationDate.Format("02-Jan-2006"))
	fmt.Println("   " + activeAipDoc.FullURLDir)

	fmt.Println("Retrieve the Navaids List")
	activeAipDoc.GetNavaids(&client)

	fmt.Println("Retrieve the Airports List")
	activeAipDoc.LoadAirports(&client)
	//activeAipDoc.DownloadAllAiportsHtmlPage(&client)
	fmt.Println("Number of identified airports: ")

	fmt.Println("Download the Airports Data")
	activeAipDoc.DownloadAllAiportsData(&client)

	//write the report JSON file
	jsonData, err := json.MarshalIndent(activeAipDoc, "", " ")
	if err != nil {
		log.Println(err)
	}
	_ = ioutil.WriteFile("info.json", jsonData, 0644)
}

/**
 * initClient inits an http client to connect to the website  by sending the
 * data to the formular.
 */
func (jpd *JpData) InitClient() http.Client {

	frmData := jpd.LoginData
	//Create a cookie Jar to manage the login cookies
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		log.Fatal(err)
	}

	/*
		//The certificate is signed by SECOM, but unable to transform the certificate to PEM.
		// It seems there is no pb on windows (maye be because the certificate has been accepted)
		//As a consequence, the verify is skip
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}

		var client = http.Client{Jar: jar, Transport: tr}
	*/
	var client = http.Client{Jar: jar}
	//login to the page
	v := url.Values{"formName": {frmData.FormName},
		"password": {frmData.Password},
		"userID":   {frmData.UserID}}

	//connect to the website
	resp, err := client.PostForm(JapanAis.LoginPage, v)
	if err != nil {
		log.Println("If error due to certificate problem, install ca-certificates")
		log.Fatal(err)
	}

	defer resp.Body.Close()
	return client
}
