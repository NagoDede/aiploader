package japan

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/NagoDede/aiploader/generic"
)

// Here's the worker, of which we'll run several
// concurrent instances. These workers will receive
// work on the `jobs` channel and send the corresponding
// results on `results`. We'll sleep a second per job to
// simulate an expensive task.
func worker(id int, url string, client *http.Client, jobs chan *generic.PdfData) {

	for j := range jobs {

		mainUrl := url + j.Link
		status := downloadPDF(mainUrl, j.FilePath, client)
		j.DownloadStatus = status

		j.ParentAirport.Wg.Done() //set the task done in the airport working group
		j.ParentAirport.NbDownloaded = j.ParentAirport.NbDownloaded + 1
		fmt.Printf("%s downloaded %d / %d \n", j.ParentAirport.Icao, j.ParentAirport.NbDownloaded, len(j.ParentAirport.PdfData))

	}
}

func downloadPDF(url string, pathFile string, client *http.Client) bool {

	//create the directory
	os.MkdirAll(filepath.Dir(pathFile), os.ModePerm)

	newFile, err := os.Create(pathFile)
	if err != nil {
		log.Fatal(err)
	}
	defer newFile.Close()

	// HTTP GET request
	response, err := client.Get(url)
	defer response.Body.Close()

	// Write bytes from HTTP response to file.
	// response.Body satisfies the reader interface.
	// newFile satisfies the writer interface.
	// That allows us to use io.Copy which accepts
	// any type that implements reader and writer interface
	numBytesWritten, err := io.Copy(newFile, response.Body)
	if err != nil {

		log.Fatal(err)
	}
	log.Printf("Downloaded %d byte file %s.\n", numBytesWritten, pathFile)
	return true
}

// DownloadAndMergeAiportData will donwload the aiport pdf files (description and charts).
// In order to save time, will download only the files that are not up to date or not created before.
// DonwloadAndMergeAirportsData has the capability to retrieve and restart a donwload if:
// - the directory where the data are stored exists but the date is not in accordance with the effective date
// - by force
// - indicated files or directory do not exist or the dates are not in accordance with the effective date
// DownloadAndMergeAirportData does not download directly the files. Instead it puts the download files
// in the jobs channel. By this way it is possible to limit more easily the number of http client used to
// download the data.
// After download, the pdf data files are merged together in order to create _full pdf file and _chart pdf file.
// If for any reason the merge fails (mainly for file problem), a new download is performed for all the airport data.
// This new download is done only one time.
func DownloadAndMergeAiportData(apt *generic.Airport, jobs *chan *generic.PdfData, docWg *sync.WaitGroup, force bool) {

	//reset the number of pdf files downloaded
	apt.NbDownloaded = 0
	//Ensures that we try at worst two times the download
	if apt.DownloadCount > 1 {
		//fmt.Println("*******" + apt.icao + " cannot perform the Merge process effciently - stop")
		log.Fatal("*******" + apt.Icao + " cannot perform the Merge process effciently - stop")
	}

	DownloadAiportData(apt, jobs, force)
	//wait the waiting group of the airport
	apt.Wg.Wait()

	//merge the pdf data if everything was done
	//thanks the Wait, the call of DetermineIsDownloaded is not mandatory.
	//But it provides a complementary means of verification$
	//Merge only if there is more than one file.
	if apt.DetermmineIsDownloaded() {
		fmt.Println("Airport: " + apt.Icao + " all docs downloaded confirmed")
		if len(apt.PdfData) > 1 {
			fmt.Printf("     Airport: %s merging files (%d). \n", apt.Icao, len(apt.PdfData))
			err := MergePdfDataOfAiport(apt)
			if err != nil {
				fmt.Printf("     Problem on Airport: %s download again \n", apt.Icao)
				DownloadAndMergeAiportData(apt, jobs, docWg, true)
			} else {
				//All the airport downloads and merge have been done. The airport can be remove of the waiting group
				docWg.Done()
			}
		} else if len(apt.PdfData) == 1 {
			//copy the file in the merge directory
			outPath := apt.AipDocument.DirMergeFiles()
			outFullMerge := generic.MergedData{FileName: apt.Icao + "_full.pdf", FileDirectory: outPath}
			opth := filepath.Join(outFullMerge.FileDirectory, outFullMerge.FileName)
			_, err := Copy(apt.PdfData[0].FilePath, opth)
			if err != nil {
				fmt.Printf("     Problem with Airport: %s unable to copy in %s \n", apt.Icao, opth)
				fmt.Println("       Download file(s) again")
				DownloadAndMergeAiportData(apt, jobs, docWg, true)
			} else {
				apt.MergePdf = append(apt.MergePdf, outFullMerge)
				docWg.Done()
			}

		} else {
			log.Printf("No PDF file for %s \n", apt.Icao)
		}
	} else {
		//all the files have not been downloaded. Start again the download...
		fmt.Println("*******" + apt.Icao + " is not completed. No PDF merge done. Start a New download")
		DownloadAndMergeAiportData(apt, jobs, docWg, true)
	}

}

func DownloadAiportData(apt *generic.Airport, jobs *chan *generic.PdfData, force bool) {

	di, err := os.Stat(apt.DirDownload())
	//determine if the target directory exists, or was created before the effective date.
	// Also, if force is done, all the files will be donwloaded again.
	if force || os.IsNotExist(err) || di.ModTime().Before(apt.AipDocument.Document().EffectiveDate) {
		//create the directory
		os.MkdirAll(apt.DirDownload(), os.ModePerm)
		//the directory does not exist or is not up to date
		apt.SetPdfDataListInChannel(jobs)

		//set the directory time to the cuurent date
		if err := os.Chtimes(apt.DirDownload(), time.Now(), time.Now()); err != nil {
			log.Fatal(err)
			panic(err)
		}
	} else if err == nil {
		//if directory exists, then case by case in regard of the file description
		for i := range apt.PdfData {
			apt.PdfData[i].ParentAirport = apt
			filePth := apt.PdfData[i].FilePath
			fi, err := os.Stat(filePth)
			if os.IsNotExist(err) {
				//the files does not exist, download it

				*jobs <- &(apt.PdfData[i])
				apt.Wg.Add(1) //add to the working group
			} else if err == nil {
				//the file exists, check if the file is before the effectiveDate.
				//As there is one directory by effective Date, there is no specific
				//ned to check if the file is after the next effective date.
				//This check is only to be sur that the directory is well up to date
				if fi.ModTime().Before(apt.AipDocument.Document().EffectiveDate) {
					*jobs <- &(apt.PdfData[i])
					apt.Wg.Add(1) //add to the working group
				} else {
					apt.PdfData[i].DownloadStatus = true
					apt.NbDownloaded = apt.NbDownloaded + 1
				}

			} else if err != nil {
				//there is an error... lets go for a panic
				log.Fatal(err)
				panic(err)
			}
		}
	} else {
		//there is an error with the directory. Lets go for a panic
		log.Fatal(err)
		panic(err)
	}
}

func Copy(src string, dst string) (int64, error) {
	src_file, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer src_file.Close()

	src_file_stat, err := src_file.Stat()
	if err != nil {
		return 0, err
	}

	if !src_file_stat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	dst_file, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer dst_file.Close()
	return io.Copy(dst_file, src_file)
}
