package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/jlaffaye/ftp"
)

type FtpInfo struct {
	Host      string
	User      string
	Password  string
	Directory string
}

func (ftpi *FtpInfo) LoadJsonFile(path string) {
	// Open our jsonFile
	jsonFile, err := os.Open(path)
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}

	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)

	err = json.Unmarshal(byteValue, ftpi)
	if err != nil {
		fmt.Println("error:", err)
	}
}

func NewFtpClient(ftpi *FtpInfo) (*ftp.ServerConn, error) {

	client, err := ftp.Dial(ftpi.Host)
	if err != nil {
		return nil, err
	}

	if err := client.Login(ftpi.User, ftpi.Password); err != nil {
		return nil, err
	}

	return client, nil
}

func SendFtpFile(c *ftp.ServerConn, inPath string, outPath string) {
	inFile, err := os.Open(inPath)
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}

	err = c.Stor(filepath.ToSlash(outPath), inFile)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(inPath + " uploaded in " + outPath)
	}

	defer inFile.Close()
}

func DisconnectFromFtpServer(c *ftp.ServerConn) {
	if err := c.Quit(); err != nil {
		log.Fatal(err)
	}
}

func DeleteFtpDirectory(c *ftp.ServerConn, dirPath string) {
	if err := c.RemoveDirRecur(dirPath); err != nil {
		log.Printf("** !! Unable to remove %s \n", dirPath)
	}
}

func CreateFtpDirectory(c *ftp.ServerConn, dirPath string) {

	if err := c.MakeDir(dirPath); err != nil {
		log.Fatal(err)
	}
}
