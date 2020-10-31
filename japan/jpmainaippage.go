package japan

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type AipDocs []*JpAipDocument

func getAipDocuments(cl *http.Client) AipDocs {
	resp, err := cl.Get(JapanAis.MainAipPage)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	availableDocs := getActiveAipDocument(resp.Body)
	return availableDocs
}

/**
 * getActiveAipDocument identifies the documents from the AIP main page.
 * It returns a table of AipDocuments
 */
func getActiveAipDocument(mainaip io.ReadCloser) AipDocs {
	var aipDocs = AipDocs{}

	doc, err := goquery.NewDocumentFromReader(mainaip)
	if err != nil {
		fmt.Println("No url found")
		log.Fatal(err)
	}

	var tempEffectiveDate time.Time
	doc.Find("table").Each(func(index int, tablehtml *goquery.Selection) {
		//The references of the documents are recorded in the tables with class Table-all-0-left
		if tablehtml.HasClass("Table-all-0-left") {
			//run across the rows of the relevant tables
			//relevant rows are identified as odd-row or even-row
			tablehtml.Find("tr").Each(func(indextr int, rowhtml *goquery.Selection) {
				if rowhtml.HasClass("odd-row") || rowhtml.HasClass("even-row") {
					var currentDate time.Time
					var effectiveDate time.Time
					var publicationDate time.Time
					var partialURL string
					var err error
					aipdoc := JpAipDocument{}

					//run across the cells
					rowhtml.Find("td").Each(func(indexth int, tablecell *goquery.Selection) {
						//cell contain the current tag, we retrieve the effective date from the if
						//The content of the cell "effective" cell will be used to confirm this date
						if tablecell.HasClass("current") {
							tempCurrent, exist := tablecell.Find("span").Attr("id")
							if exist {
								cleanStr := tempCurrent[len("efct-"):]
								currentDate, err = buildDateFromYYYYMMDD(cleanStr)
								if err != nil {
									fmt.Printf("2-Unable to build date from %s \n", cleanStr)
									panic(err)
								}
							}
						}

						if tablecell.HasClass("date") && !tablecell.HasClass("td-right-top-0-0 date") {
							effectiveDate, err = buildDateFromDD_MMM_YYYY(tablecell.Text())
							if err != nil {
								fmt.Printf("1-Unable to build date from %s \n", tablecell.Text())
								panic(err)
							}

							//retrieve the address
							tempURL, exist := tablecell.Find("a").Attr("href")
							if exist {
								partialURL = tempURL
							} else {
								fmt.Printf("3 - Unable to retrieve URL \n")
							}
						}

						if tablecell.HasClass("td-right-top-0-0 date") {
							publicationDate, err = buildDateFromDD_MMM_YYYY(tablecell.Text())
							if err != nil {
								fmt.Printf("4-Unable to build date from %s \n", tablecell.Text())
								panic(err)
							}
						}
					})

					//after review of the cells, there is enough data to create an AipDocument
					//create the aipdoc
					aipdoc.EffectiveDate = effectiveDate
					aipdoc.PublicationDate = publicationDate
					aipdoc.PartialURL = partialURL
					aipdoc.FullURLPage = JapanAis.MainAipActiveURL + partialURL
					u := strings.LastIndex(aipdoc.FullURLPage, "/")
					aipdoc.FullURLDir = aipdoc.FullURLPage[:u+1]

					//identify the most recent but applicable document
					if effectiveDate.Before(time.Now()) {
						if effectiveDate.After(tempEffectiveDate) {
							tempEffectiveDate = effectiveDate
						}
					}

					//confirm that dates are coherent
					//the current date (used for the identification by a dot in the table by using javascript) and the
					//effective date shall be the same
					if effectiveDate.Equal(currentDate) {
						aipdoc.IsValidDate = true
					} else {
						aipdoc.IsValidDate = false
					}

					//confirm the url is in accordance with the dates
					//the url shall be publicationdate/eAip/effectiveDate
					//we retrieve the dates from the Url and compare with the extracted data
					pubDateURL, err := getPublicationDateFromPartialURL(partialURL)
					if err != nil {
						fmt.Printf("6-Unable to get Publication date from %s \n", partialURL)
						panic(err)
					}
					effDateURL, err := getEffectiveDateFromPartialURL(partialURL)
					if err != nil {
						fmt.Printf("7-Unable to get Effective date from %s \n", partialURL)
						panic(err)
					}
					if effectiveDate.Equal(effDateURL) && publicationDate.Equal(pubDateURL) {
						aipdoc.IsPartialURLValid = true
					} else {
						aipdoc.IsPartialURLValid = false
					}

					//add the aipDoc to the list
					aipDocs = append(aipDocs, &aipdoc)
				}

			})
		}
	})

	//setActiveAipDoc(aipDocs, tempEffectiveDate)
	aipDocs.setActiveAipDoc(tempEffectiveDate)
	//aipDocs.printAipDocs()

	return aipDocs
}

/***
 * Set an AIP document as active in regard of the targetDate.
 * If no or more than one document have been identified, create a panic
 */
func (docs *AipDocs) setActiveAipDoc(targetDate time.Time) {
	var countActive int
	fmt.Printf("Target Date is: %s \n", targetDate)
	var activeDocs AipDocs

	for _, aipdoc := range *docs {
		if aipdoc.EffectiveDate.Equal(targetDate) {
			//This will identify the document having an effective date in accordance with the target date.
			//several documents could be identified as active
			if aipdoc.IsPartialURLValid && aipdoc.IsValidDate {
				aipdoc.IsActive = true
				countActive = countActive + 1
				activeDocs = append(activeDocs, aipdoc)
			}
		}
	}
}

func (docs *AipDocs) getActiveAipDoc() *JpAipDocument {
	var activeDocs []*JpAipDocument

	//count the number of active document
	//only one document is active
	var counter int
	var activeDoc *JpAipDocument //*generic.AipDocument
	for _, aipdoc := range *docs {
		if aipdoc.IsActive {
			counter++
			activeDoc = aipdoc
			activeDocs = append(activeDocs, activeDoc)
		}
	}

	if counter == 1 {
		return activeDoc
	}

	if counter == 0 {
		panic("No document identified as Active")
	}

	//for the other cases, need to sort by publication date
	//the active documents to retrieve the most recent document
	sort.SliceStable(activeDocs, func(i, j int) bool {
		return activeDocs[i].PublicationDate.After(activeDocs[j].PublicationDate)
	})

	activeDoc = activeDocs[0]

	for i, d := range *docs {
		if i > 0 {
			d.IsActive = false
		}
	}
	fmt.Printf("Selected Active document is effective date: %s - publication date %s \n", activeDoc.EffectiveDate, activeDoc.PublicationDate)
	return activeDoc
}

func (docs AipDocs) printAipDoc() {
	for _, doc := range docs {
		fmt.Println("Effective Date")
		fmt.Println(doc.EffectiveDate)
		fmt.Println(doc.IsActive)
	}
}

/**
 * buildDateFromDD_MMM_YYYY build a date following a string with the format
 * DD MMM YYYY (i.e. 14 jul 2019, 1 Mar 2018)
 */
func buildDateFromDD_MMM_YYYY(s string) (time.Time, error) {
	strTable := strings.Split(s, " ")
	day := strTable[0]
	if len(day) == 1 {
		day = "0" + day
	}
	month := strTable[1]
	year := strTable[2]
	dyear, _ := strconv.Atoi(year)
	dyear = dyear - 2000

	var str strings.Builder
	str.WriteString(day)
	str.WriteString(" ")
	str.WriteString(month)
	str.WriteString(" ")
	str.WriteString(strconv.Itoa(dyear))
	str.WriteString(" 00:00 UTC")

	date, err := time.Parse(time.RFC822, str.String())

	if err != nil {
		return time.Now(), errors.New("Unable to convert " + s + " to a date. Confirm the format is DD MMM YYYY (14 jul 2019)")
	}

	return date, nil
}

/**
 * buildData builds a date (time.Time), initiliazed at 0:0:0Z from a date with the YYYYMMDD date
 */
func buildDateFromYYYYMMDD(s string) (time.Time, error) {
	efctYear := s[0:4]
	efctMth := s[4:6]
	efctDay := s[6:8]
	date, err := buildDate(efctYear, efctMth, efctDay)
	if err != nil {
		return time.Now(), errors.New("Unable to convert " + s + " to a date. Confirm format is YYYYMMDD")
	}
	return date, nil
}

/***
 * builDate by giving a year (YYYY), month (digit MM), day (DD)
 */
func buildDate(year string, month string, day string) (time.Time, error) {
	dyear, erry := strconv.Atoi(year)
	if erry != nil {
		return time.Now(), errors.New("Year format is not recognized or managed. Shall be YYYY (2019, 2020)")
	}
	dmonth, errm := strconv.Atoi(month)
	if errm != nil {
		return time.Now(), errors.New("Month format is not recognized or managed. Shall be MM (01..12)")
	}

	dday, errd := strconv.Atoi(day)
	if errd != nil {
		return time.Now(), errors.New("Day format is not recognized or managed. Shall be MM (01..31)")
	}
	return time.Date(dyear, time.Month(dmonth), dday, 0, 0, 0, 0, time.UTC), nil
}

/***
 * getEffectiveDateFromPartialUrl return the effective date from the partial Url
 * Partial Url follows the schem: /publicationdate/eAIP/effectiveDate/index.html
 * The publication and effective dates are in the format YYYYMMDD.
 */
func getEffectiveDateFromPartialURL(pth string) (time.Time, error) {
	strTable := strings.Split(pth, "/")

	date, err := buildDateFromYYYYMMDD(strTable[2])
	if err != nil {
		return time.Now(), errors.New("Unable to extract and retrieve a valide effective date in the URL. Read: " + strTable[2] + " expect YYYYMMDD format.")
	}
	return date, nil
}

/***
 * getPublicationDateFromPartialUrl return the effective date from the partial Url
 * Partial Url follows the schem: /publicationdate/eAIP/effectiveDate/index.html
 * The publication and effective dates are in the format YYYYMMDD.
 */
func getPublicationDateFromPartialURL(pth string) (time.Time, error) {
	strTable := strings.Split(pth, "/")

	date, err := buildDateFromYYYYMMDD(strTable[0])
	if err != nil {
		return time.Now(), errors.New("Unable to extract and retrieve a valide publication date in the URL. Read: " + strTable[0] + " expect YYYYMMDD format.")
	}
	return date, nil
}

func (docs AipDocs) GetNextDate(actDoc JpAipDocument) time.Time {

	sort.SliceStable(docs, func(i, j int) bool {
		return docs[i].EffectiveDate.Before(docs[j].EffectiveDate)
	})

	for _, d := range docs {
		if d.EffectiveDate.After(actDoc.EffectiveDate) {
			fmt.Printf("Next date is %s \n", d.EffectiveDate)
			return d.EffectiveDate
		}
	}
	log.Panic("Unable to identify the next date")
	return time.Now()
}
