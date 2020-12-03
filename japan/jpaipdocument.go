package japan

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/NagoDede/aiploader/generic"
	"github.com/PuerkitoBio/goquery"
)

type JpAipDocument struct {
	generic.AipDocument
	Airports []JpAirport
}

func (aipdcs *JpAipDocument) GetNavaids(cl *http.Client) []generic.Navaid {
	var indexUrl = aipdcs.FullURLDir + JapanAis.AipIndexPageName
	fmt.Println("   Retrieve RadioNavigation  in " + indexUrl)
	resp, err := cl.Get(indexUrl)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		fmt.Println("No url found for navaid extraction")
		log.Fatal(err)
	}

	var navaidpage string
	doc.Find(`div[id="ENR-4details"]`).Each(func(index int, divhtml *goquery.Selection) {
		divhtml.Find(`div[class="H3"]`).Each(func(index int, ahtml *goquery.Selection) {
			t, titleEx := ahtml.Find("a").Attr("title")
			if titleEx {
				if strings.Contains(t, "NAVIGATION AIDS") {
					href, hrefEx := ahtml.Find("a").Attr("href")
					if hrefEx {
						navaidpage = href
						fmt.Println("Page to the Radio Navigation" + href)
					}
				}
			}
		})
	})

	fmt.Println("Retrieve data from " + aipdcs.FullURLDir + navaidpage)
	navaidresp, err := cl.Get(aipdcs.FullURLDir + navaidpage)
	if err != nil {
		log.Fatal(err)
	}

	defer navaidresp.Body.Close()
	navaidsdoc, err := goquery.NewDocumentFromReader(navaidresp.Body)
	if err != nil {
		fmt.Println("No url found for navaid extraction")
		log.Fatal(err)
	}

	navaids, trCount := loadNavaidsFromHtmlDoc(navaidsdoc)
	//confirm we have the same number
	if trCount == len(navaids) {
		return nil
	} else {
		log.Println("Number of rows in the table and identified Navaids differs")
		return nil
	}
}

func loadNavaidsFromHtmlDoc(navaidsdoc *goquery.Document) (map[string]generic.Navaid, int) {
	//navs := //[]Navaid{}
	var navs = make(map[string]generic.Navaid)
	trCount := 0
	navaidsdoc.Find(`table`).Each(func(index int, divhtml *goquery.Selection) {
		tbody := divhtml.Find(`tbody`).First()
		trCount = 0
		tbody.Find("tr").Each(func(index int, tr *goquery.Selection) {
			id, titleEx := tr.Attr("id")
			if titleEx {
				fmt.Println(id)
				if strings.HasPrefix(id, "NAV-") {
					nav := generic.Navaid{}
					nav.SetFromHtmlSelection(tr)
					if nav.Key != "" {
						if val, ok := navs[nav.Key]; ok {
							log.Printf("%s appears several time", val.Key)
						} else {
							navs[nav.Key] = nav
							trCount++
						}
					} else {
						log.Printf("%s is disregarded - not NAV data", id)
					}
				} else {
					log.Printf("%s is disregarded - not NAV data \n", id)
				}
			}
		})
	})

	return navs, trCount
}

/***
 * Retrieves the Location Indicators codes (ICAO codes) in a map.
 * The map key is the ICAO code while the value is the location name.
 * Use the information provided in the GEN AIP.
 * For Japan, somes ICAO codes are not airport or heliport. They are associated to NOTAMs.
 * These codes are identified by an asterix (*). The asterix is removed and the code is recorded.
 ***/
func (aipdcs *JpAipDocument) LoadLocationIndicators(cl *http.Client) *map[string]string {
	var indexUrl = aipdcs.FullURLDir + JapanAis.LocationCodePage
	fmt.Println("   Retrieve Location indicatores from: " + indexUrl)
	resp, err := cl.Get(indexUrl)
	defer resp.Body.Close()

	if err != nil {
		fmt.Printf("Problem while reading %s \n", indexUrl)
		log.Fatal(err)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		fmt.Println("No url found for airports extraction")
		log.Fatal(err)
	}

	locationCodes := make(map[string]string)

	doc.Find(`td[class="colsep-1"]`).Find(`tr[id^="ICAO-"]`).Each(func(index int, divhtml *goquery.Selection) {
		var tds = divhtml.ChildrenFiltered(`td`)
		var location = tds.Eq(0).Text()
		var code = tds.Eq(1).Text()

		if location != "" && code != "" && len(code)>=4 {
			locationCodes[code[0:4]] = strings.Replace(location,"\n", " ",-1)
		}
	})

	return &locationCodes
}

func (aipdcs *JpAipDocument) LoadAirports(cl *http.Client) {
	var indexUrl = aipdcs.FullURLDir + JapanAis.AipIndexPageName

	fmt.Println("   Retrieve Airports list from: " + indexUrl)
	resp, err := cl.Get(indexUrl)
	defer resp.Body.Close()
	if err != nil {
		fmt.Println("Problem while reading %s \n", indexUrl)
		log.Fatal(err)
	} else {

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			fmt.Println("No url found for airports extraction")
			log.Fatal(err)
		} else {

			var countWkr int
			var wg sync.WaitGroup
			doc.Find(`div[id="AD-2details"]`).Each(func(index int, divhtml *goquery.Selection) {
				divhtml.Find(`div[class="H3"]`).Each(func(index int, h3html *goquery.Selection) {
					countWkr = countWkr + 1
					fmt.Println("Main: Starting worker", countWkr)
					wg.Add(1)
					go aipdcs.retrieveAirport(&wg, h3html, cl)
				})
			})

			fmt.Println("Main: Waiting for workers to finish")
			wg.Wait()
			fmt.Println("Main: Completed")
		}
	}
}

func (aipDoc *JpAipDocument) retrieveAirport(wg *sync.WaitGroup, h3html *goquery.Selection, cl *http.Client) {
	defer wg.Done()
	h3html.Find("a").Each(func(index int, ahtml *goquery.Selection) {
		idAd, exist := ahtml.Attr("title")
		if exist {
			if strings.Contains(idAd, "AERO") || strings.Contains(idAd, "aero") {
				idId, idEx := ahtml.Attr("id")
				if idEx {
					ad := JpAirport{}
					ad.AipDocument = aipDoc

					ad.Icao = idId[5:9]
					ad.Title = ahtml.Text()[7:]
					href, hrefEx := ahtml.Attr("href")
					if hrefEx {
						ad.Link = href
					}
					ad.PdfData = []generic.PdfData{}
					fmt.Println(ad.Icao)
					fmt.Println(ad.Title)
					ad.DownloadPage(cl)
					ad.GetPDFFromHTML(cl, aipDoc.FullURLDir)
					//maps, i := ad.GetNavaids()
					aipDoc.Airports = append(aipDoc.Airports, ad)
				}
			}
		}
	})
}

func (aipDoc *JpAipDocument) DownloadAllAiportsHtmlPage(cl *http.Client) {
	var docsWg sync.WaitGroup
	for i, _ := range aipDoc.Airports {
		docsWg.Add(1)
		apt := &aipDoc.Airports[i]
		apt.AipDocument = aipDoc
		generic.DownloadAirportPageSync(cl, &docsWg, apt)
	}
	docsWg.Wait()
}

func (aipDoc *JpAipDocument) DownloadAllAiportsData(client *http.Client) {
	jobs := make(chan *generic.PdfData, 10)

	var w int
	var docsWg sync.WaitGroup
	for i, _ := range aipDoc.Airports {
		docsWg.Add(1)

		apt := &aipDoc.Airports[i]
		apt.AipDocument = aipDoc //refresh the pointer (case we miss something)
		//create the workers. the number is limited by 5 at the time being
		if w < 5 {
			w = w + 1
			go worker(w, aipDoc.FullURLDir, client, jobs)
		}

		DownloadAndMergeAiportData(&apt.Airport, &jobs, &docsWg, false)
	}
	docsWg.Wait()

	fmt.Println("Download and merge - done")

}
