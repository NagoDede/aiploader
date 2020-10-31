package generic

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

/*
 The Airport Type contains the information for the definition of an airport in the AIP.
 The ICAO code is the main mean of identification of the airport.
 The individual charts are recorded in the PdfData tables.
 In order to manage the downloads, the structure contains information about the status of the downloads.
 A waiting group is associated to the Airport structure in order to manage the downloads
 or any other tasks associated to the airport.
*/
type Airport struct {
	Title       string
	Icao        string
	Link        string `json:"-"`
	AirportType string `json:"-"`
	DownloadData
	AdminData   AdminData
	Navaids     map[string]Navaid
	PdfData     []PdfData    `json:"-"`
	MergePdf    []MergedData `json:"-"`
	Com         []ComData
	//Airport     IAirport `json:"-"`
	AipDocument IAipDocument     `json:"-"`
	HtmlPage    string           `json:"-"`
}

type IAirport interface {
	GetPDFFromHTML(cl *http.Client, aipURLDir string)
	DownloadPage(cl *http.Client)
	GetNavaids() (map[string]Navaid, int)
}

type DownloadData struct {
	DownloadCount int
	Wg            sync.WaitGroup
	NbDownloaded  int
}

/*
AdminData contains the admnistrative information of the airport.
Basic information are related to the ARP coordinates, elevation, magnetic variations,...
*/
type AdminData struct {
	ArpCoord         string
	Elevation        string
	Mag_var          string
	Mag_annualchange string
	Geoid_undulation string
	Traffic_types    string
}

/*
 ComData describes the communication means available on the airport.
*/
type ComData struct {
	service         string
	frequency       string
	callSign        string
	operationsHours string
	remarks         string
}

type PdfData struct {
	ParentAirport   *Airport
	Title           string
	DataContentType string
	Link            string
	FileName        string
	FilePath        string
	DownloadStatus  bool
}

type MergedData struct {
	//ParentAirport *Airport
	Title         string
	FileDirectory string
	FileName      string
}


/*
	Get the download directory.
*/
func (a *Airport) DirDownload() string {
	return filepath.Join(a.AipDocument.DirMainDownload(), a.Icao)
}

func (a *Airport) AddPdfData(pdf PdfData)  {
	pdf.FilePath = filepath.Join(a.DirDownload(), pdf.FileName)
	a.PdfData = append(a.PdfData, pdf)
}

/*
Set in channel the content of the Airport.PdfData in the indicated channel.
Also, add each PdfData file as a task in the Waiting group of the Airport
*/
func (a *Airport) SetPdfDataListInChannel(jobs *chan *PdfData) {
	for i := range a.PdfData {
		a.PdfData[i].ParentAirport = a
		*jobs <- &(a.PdfData[i])
		a.Wg.Add(1) //add to the working group
	}
}

/*
	Determine if all airport's data have been downloaded.
*/
func (a *Airport) DetermmineIsDownloaded() bool {
	var tempB bool
	tempB = true
	for _, pdf := range a.PdfData {
		tempB = tempB && pdf.DownloadStatus
	}
	if tempB {
		a.DownloadCount = a.DownloadCount + 1
	}
	return tempB
}

/*
	Download the airport page in a synchronous way.
*/
func  DownloadAirportPageSync(cl *http.Client, docWg *sync.WaitGroup, a IAirport) {
	go a.DownloadPage(cl)
	docWg.Done()
}

/*
	Determine if the airport web page shall be downloaded.
	Return true if the page shall be downloaded.
	By default, we download the page.
*/
func (apt *Airport) ShouldIDownloadHtmlPage(realPath string, bodySize int64) bool {
	if st, err := os.Stat(realPath); err == nil {
		//The file exists, check the date of the file
	
		if st.ModTime().After(apt.AipDocument.Document().EffectiveDate) && st.ModTime().Before(apt.AipDocument.Document().NextEffectiveDate) {
			if bodySize == st.Size() {
				return false
			}
		}
		return true

	} else if os.IsNotExist(err) {
		return true
	} else {
		// Schrodinger: file may or may not exist. See err for details.
		// Therefore, do *NOT* use !os.IsNotExist(err) to test for file existence
		log.Printf("File %s is not writeable or readable \n", realPath)
		return true
	}
	return true
}
