package generic

import (
	"fmt"
	"net/http"
	"path/filepath"
	"time"
)

type AipDocument struct {
	IsActive          bool
	EffectiveDate     time.Time
	PublicationDate   time.Time
	NextEffectiveDate time.Time
	ProcessDate       time.Time		
	IsValidDate       bool
	PartialURL        string
	IsPartialURLValid bool
	FullURLDir        string
	FullURLPage       string
	Airports          []Airport
	Navaids			  []Navaid
	CountryCode       string
}

type IAipDocument interface {
	LoadAirports(cl *http.Client) 
	GetNavaids(cl *http.Client) []Navaid
	DownloadAllAiportsData(client *http.Client)
	DownloadAllAiportsHtmlPage(cl *http.Client)
	DirMainDownload() string
	DirMergeFiles() string
	Document() AipDocument
}

func (aip *AipDocument) DirMainDownload() string {
	dir := filepath.Join(ConfData.MainLocalDir, aip.CountryCode)
	t := aip.EffectiveDate
	dateDir := fmt.Sprintf("%d%02d%02d", t.Year(), t.Month(), t.Day())
	return filepath.Join(dir, dateDir)
}

func (aip *AipDocument) DirMergeFiles() string {
	return filepath.Join(aip.DirMainDownload(), ConfData.MergeDir)
}

func (aip *AipDocument) Document() AipDocument {
	return *aip
}



