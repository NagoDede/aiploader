package japan

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"io"

	"path/filepath"

	"github.com/PuerkitoBio/goquery"
	"github.com/NagoDede/aiploader/generic"

)

type JpAirport struct {
	generic.Airport
}
 
/*
	Download the AIP webpage of the airport.
	This webpage will be used to retrieve all the relevant information.
	The path to the downloaded file will be indicated in the airport.htmlPage field.
*/
func (apt *JpAirport) DownloadPage(cl *http.Client) { //, aipURLDir string) {

	var indexURL = apt.AipDocument.Document().FullURLDir + apt.Link // aipURLDir + apt.link
	fmt.Println("     Download the airport page: " + indexURL)
	resp, err := cl.Get(indexURL)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	// HTTP GET request

	filePth := filepath.Join(apt.DirDownload(), apt.Icao+".html")

	if apt.ShouldIDownloadHtmlPage(filePth, resp.ContentLength) {
		//create the directory
		os.MkdirAll(apt.DirDownload(), os.ModePerm)
		newFile, err := os.Create(filePth)
		// Write bytes from HTTP response to file.
		// response.Body satisfies the reader interface.
		// newFile satisfies the writer interface.
		// That allows us to use io.Copy which accepts
		// any type that implements reader and writer interface

		numBytesWritten, err := io.Copy(newFile, resp.Body)
		if err != nil {
			log.Printf("Unable to write the webpage %s in directory %s \n", indexURL, filePth)
			log.Fatal(err)
		}
		log.Printf("Airport %s - downloaded %d byte file %s.\n", apt.Icao, numBytesWritten, filePth)
	} else {
		log.Printf("Airport %s - page %s not saved, local copy is good %s.\n", apt.Icao, indexURL, filePth)
	}
	apt.HtmlPage = filePth
}

// GetPDFFromHTML will retrieve the PDF information (which will be downloaded later) in a HTML
// indicated by combination of the fullURLDir and the content of the Airport.link.
// The function will populate the PdfData table of the Airport.
// This approach allows a simple way to conbsider the fact that the main directory evolves for each new AIP vesion.
// As there is the web pages do not contain a direct link to the main PDF file, a dedicated entry
// is done during the process.
// There is no need to sort the identified PDF files. The natural sorting, done by the data recovery ensures
// the correct order. The name of the files is not sufficient to set them in the corect order
func (apt *JpAirport) GetPDFFromHTML(cl *http.Client, aipURLDir string) {

	apt.DownloadCount = 0 //reinit the download counter
	var indexUrl = aipURLDir + apt.Link
	divWord := `div[id="` + apt.Icao + "-AD-2.24" + `"]`

	fmt.Println("     Retrieve PDF pathes from: " + indexUrl)
	resp, err := cl.Get(indexUrl)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		fmt.Println("No url found for airports extraction")
		log.Fatal(err)
	}

	//create and retrieve the main PDF page
	//the order shall be respected, else the page sequence could not be respected during the merge process
	//So first, it is the text/description pdf
	apt.AddPdfData(apt.getTxtPDFFile())
	//apt.PdfData = append(apt.PdfData, apt.getTxtPDFFile())

	doc.Find(divWord).Each(func(index int, divhtml *goquery.Selection) {
		divhtml.Find("a").Each(func(index int, ahtml *goquery.Selection) {
			pdfLink, ext := ahtml.Attr("href")
			if ext {
				apt.AddPdfData(apt.getChartPDFFile(pdfLink))
				//apt.PdfData = append(apt.PdfData, apt.getChartPDFFile(pdfLink))
			}

		})
	})
}

// mainPDFFile creates the path to the main PDF as there is no associated link in the webpage
// and provides it in a PdfData structure (dataContentType is associated to Text)
func (apt *JpAirport) getTxtPDFFile() generic.PdfData {
	pdfTxt := generic.PdfData{}
	pdfTxt.ParentAirport = &apt.Airport
	pdfTxt.DataContentType = "Text"
	pdfTxt.Link = fmt.Sprintf("pdf/JP-AD-2-%s-en-JP.pdf", apt.Icao)
	pdfTxt.FileName = fmt.Sprintf("JP-AD-2-%s-en-JP.pdf", apt.Icao)

	return pdfTxt
}

func (apt *JpAirport) getChartPDFFile(partialLink string) generic.PdfData {
	pdfChart := generic.PdfData{}
	pdfChart.ParentAirport = &apt.Airport
	pdfChart.DataContentType = "Chart"
	pdfChart.Link = partialLink
	pdfChart.FileName = filepath.Base(partialLink)
	return pdfChart
}

func (apt *JpAirport) GetNavaids() (map[string]generic.Navaid, int) {

	if apt.HtmlPage == "" {
		log.Println("Html File is not downloaded")
		return nil, 0
	}
	//div[id="ENR-4details"]`
	divId := fmt.Sprintf(`div[id="%s-AD-2.19"]`, apt.Icao)
	//divId := `div=["` + apt.icao + "-AD-2.19" + `"]`

	f, err := os.Open(apt.HtmlPage)
	if err != nil {
		log.Println("Unable to open " + apt.HtmlPage)
	}
	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		fmt.Println("Unable to parse")
		log.Fatal(err)
	}

	sel := doc.Find(divId).First()
	navaids, trcount := apt.loadNavaidsFromHtmlDoc(sel)

	fmt.Println(navaids)
	fmt.Println(trcount)
	return navaids, trcount
}

func (apt *JpAirport) loadNavaidsFromHtmlDoc(div *goquery.Selection) (map[string]generic.Navaid, int) {
	//navs := //[]Navaid{}
	apt.Navaids = make(map[string]generic.Navaid)
	trCount := 0
	div.Find("table").Each(func(index int, divhtml *goquery.Selection) {
		tbody := divhtml.Find(`tbody`).First()
		trCount = 0
		tbody.Find("tr").Each(func(index int, tr *goquery.Selection) {
			aids, isok := apt.loadNavaidsFromTr(tr)
			if isok {
				apt.Navaids[aids.Key] = aids
			}
			fmt.Println(aids)
		})
	})

	return apt.Navaids, trCount
}

func (apt *JpAirport) loadNavaidsFromTr(tr *goquery.Selection) (generic.Navaid, bool) {
	var n generic.Navaid
	tr.Find("td").Each(func(index int, td *goquery.Selection) {
		switch index {
		case 0:

			n.NavaidType = strings.TrimSpace(td.Text())
			if strings.Contains(n.NavaidType, "(") {
				n.NavaidType = strings.TrimSpace(n.NavaidType[0:strings.Index(n.NavaidType, "(")])
			}
			n.MagVar = getMagVariationFromTextOfjpAirportData(td.Text())
		case 1:
			n.Id = strings.TrimSpace(td.Text())
		case 2:
			n.Frequency = strings.TrimSpace(td.Text())
		case 3:
			n.OperationsHours = strings.TrimSpace(td.Text())
		case 4:
			n.Position.Latitude = getLatitudeFromTextOfjpAirportData(td.Text())
			n.Position.Longitude = getLongitudeFromTextOfjpAirportData(td.Text())
		case 5:
			n.Elevation = strings.TrimSpace(td.Text())
		case 6:
			n.Remarks = strings.TrimSpace(td.Text())
		}

		if (n.Id != "") && (n.Id != "-") {
			n.Key = n.Id + " " + n.NavaidType
		} else {
			n.Key = apt.Icao + " " + n.NavaidType
		}

	})

	//Determine if the identifed raw is a real Navaids or the titles of the table
	//If it is title, we return false
	//To do this, we test the column where only text should be used.
	//If a number is defined, it is a title row

	_, errCol1 := strconv.Atoi(n.NavaidType)
	_, errCol2 := strconv.Atoi(n.Id)

	if (strings.Compare(n.Name, "ID") == 0) ||
		strings.Contains(n.Frequency, "requency") ||
		(errCol1 == nil) || (errCol2 == nil) ||
		(strings.Contains(n.NavaidType, "Nil")) ||
		(strings.Contains(n.Id, "Nil")) ||
		(strings.TrimSpace(n.NavaidType) == "") {
		return n, false
	} else {
		return n, true
	}
}

func getLatitudeFromTextOfjpAirportData(t string) float32 {
	latre := regexp.MustCompile(`[0-9]*\.?[0-9]+[N|S]`)
	latitude := string(latre.Find([]byte(t)))
	lat, err := generic.ConvertDDMMSSSSLatitudeToFloat(latitude)
	if err != nil {
		log.Printf("%s Latitude Conversion problem %f \n", t, lat)
		log.Println(err)
		return 0
	} else {
		return lat
	}
}

func getLongitudeFromTextOfjpAirportData(t string) float32 {
	longre := regexp.MustCompile(`[0-9]*\.?[0-9]+[E|W]`)
	longitude := string(longre.Find([]byte(t)))
	long, err := generic.ConvertDDDMMSSSSLongitudeToFloat(longitude)
	if err != nil {
		log.Printf("%s Longitude Conversion problem %f \n", t, long)
		log.Println(err)
		return 0
	} else {
		return long
	}
}

//getMagVariationFromTextOfjpAirportData extracts from a text
//the magnetic variation which is usually associated to a VOR, TACAN.
//Exemple VOR(4°W0.25W/y) --> (4°W0.25W/y)
func getMagVariationFromTextOfjpAirportData(t string) string {
	magre := regexp.MustCompile(`\((.*?)\)`)
	mag := string(magre.Find([]byte(t)))
	return strings.TrimSpace(mag)
}
