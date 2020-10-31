package japan

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/NagoDede/aiploader/generic"
	pdf "github.com/NagoDede/unipdf/model"
	"github.com/NagoDede/aiploader/writerseeker"
)

func MergePdfDataOfAiport(apt *generic.Airport) error {
	var outPath string
	pdfWriter := pdf.NewPdfWriter()
	pdf.SetPdfCreationDate(time.Now())
	pdf.SetPdfAuthor("Nagoy Dede")
	pdf.SetPdfKeywords(apt.Icao + " AIP Japan")
	pdf.SetPdfProducer("AipDownloader")

	outPath = apt.AipDocument.DirMergeFiles()

	//create the directory
	os.MkdirAll(outPath, os.ModePerm)

	outFullMerge := generic.MergedData{FileName: apt.Icao + "_full.pdf", FileDirectory: outPath}
	apt.MergePdf = append(apt.MergePdf, outFullMerge)
	outChartMerge := generic.MergedData{FileName: apt.Icao + "_chart.pdf", FileDirectory: outPath}
	apt.MergePdf = append(apt.MergePdf, outChartMerge)

	outFullPath := filepath.Join(outPath, apt.Icao+"_full.pdf")
	outChartPath := filepath.Join(outPath, apt.Icao+"_chart.pdf")

	//First create the Charts merge file
	pdf.SetPdfTitle(apt.Icao + "AIP charts ")
	pdf.SetPdfSubject(apt.Icao + " merged charts")
	for _, pdfD := range apt.PdfData[1:] {
		err := mergeInPdfWriter(&pdfWriter, &pdfD)
		if err != nil {
			return err
		}
	}

	if shouldUpdateMergePdfFile(apt, outChartPath, &pdfWriter) {
		err2 := writePdfWriter(&pdfWriter, outChartPath)
		if err2 != nil {
			return err2
		}
	}

	//create the full merge
	pdf.SetPdfTitle(apt.Icao + " AIP document")
	pdf.SetPdfSubject(apt.Icao + " merged AIP document")
	pdfFullWriter := pdf.NewPdfWriter()
	for _, pdfD := range apt.PdfData {
		err := mergeInPdfWriter(&pdfFullWriter, &pdfD)
		if err != nil {
			return err
		}
	}

	if shouldUpdateMergePdfFile(apt, outFullPath, &pdfFullWriter) {
		err2 := writePdfWriter(&pdfFullWriter, outFullPath)
		if err2 != nil {
			return err2
		}
	}

	return nil

}

func mergeInPdfWriter(pdfWriter *pdf.PdfWriter, pdfD *generic.PdfData) error {
	inPath := pdfD.FilePath
	f, err := os.Open(inPath)
	if err != nil {
		log.Println("Error during  os.Open(inPath) " + inPath)
		return fmt.Errorf("Error while opening PDF file " + inPath)
	}

	defer f.Close()

	pdfReader, err2 := pdf.NewPdfReader(f)

	if err2 != nil {
		log.Println("Error during  pdf.NewPdfReader(f) " + inPath)
		return fmt.Errorf("Error during PDFReader creation for file " + inPath)
	}
	numPages, err3 := pdfReader.GetNumPages()
	if err3 != nil {
		log.Println("Error during  pdf.GetNumPages()" + inPath)
		return fmt.Errorf("Error when retrieving the number of pages of the file " + inPath)
	}

	for i := 0; i < numPages; i++ {
		pageNum := i + 1

		page, err4 := pdfReader.GetPage(pageNum)
		if err4 != nil {
			log.Println("Error while retrieving the page %d of file %s", pageNum, inPath)
			return fmt.Errorf("Error while retrieving the page %d of file %s", pageNum, inPath)
		}

		err5 := pdfWriter.AddPage(page)
		if err5 != nil {
			log.Println("Error during  pdfWriter.AddPage(page)" + inPath)
			return fmt.Errorf("Error while adding page " + inPath)
		}
	}
	return nil
}

func writePdfWriter(pdfWriter *pdf.PdfWriter, outPath string) error {
	fWrite, err := os.Create(outPath)
	if err != nil {
		log.Println("Error during  os.Create(outPath)" + outPath)
		return fmt.Errorf("Error during  pdf creation of file " + outPath)
	}

	defer fWrite.Close()

	err = pdfWriter.Write(fWrite)
	if err != nil {
		log.Println("Error during  pdfWriter.Write(fWrite)" + outPath)
		return fmt.Errorf("Error during pdf writing " + outPath)
	}

	return nil
}

// Determine if the merge PDF file shqll be write as originFile.
// Priority is given to rewrite again the file in case of discrepency.
// It assumes that the curtrent merge file is the correct one (allows a recovery in case of crash)
// There is no need to overwrite (return false)the originFile file if:
//		- The file exists (obviously); and
//		- The file creation date is between the effectiveDate and the nextEffectiveDate; and
//		- The file as the same number of pages than the pages defined in the pdfWriter; and
//		- Size of the file is in 1% mergins
//		(don't know why, but for some files, with the same inputs file size uis slightly different)
//There is a need to overwrite the file (return true) if:
//		- The originFile doesnot exist; or
//		- The originFile cannot be opened as PDF; or
//		- There is a discrepency in the number of pages of the originFile file and the pdfWriter; or
//		- There is size discrepency between the originFile and the pdfWriter
//Uncovered case, return true
func shouldUpdateMergePdfFile(apt *generic.Airport, originFile string, pdfWriter *pdf.PdfWriter) bool {
	if st, err := os.Stat(originFile); err == nil {

		//determine if the dates are correct
		//File shall be updated if the dates are outside the effective range
		if st.ModTime().Before(apt.AipDocument.Document().EffectiveDate) || st.ModTime().After(apt.AipDocument.Document().NextEffectiveDate) {
			return true
		}

		//open the pdf file
		f, err := os.Open(originFile)
		if err != nil {
			//unable to open the file
			log.Printf("Unable to open the file %s, write the file again \n ", originFile)
			return true
		}
		defer f.Close()
		realPdf, err2 := pdf.NewPdfReader(f)
		var realPages int
		if err2 != nil {
			log.Printf("Unable to open the file %s as PDF file, write the file again \n ", originFile)
			return true
		}
		numPages, err := realPdf.GetNumPages()
		if err != nil {
			log.Printf("Unable to retrieve the number of pages of %s write the file again \n ", originFile)
			return true
		}
		realPages = numPages

		//simulate a write of the document in order to get the file size and the number of pages
		writerSeeker := &writerseeker.WriterSeeker{}
		size, err := pdfWriter.WriteGetSize(writerSeeker) //Discard)
		if err != nil {
			log.Printf("Unable to write in the buffer for %s", originFile)
			//unable to write in the buffer, just keep the current file, do not request an upload
			return false
		}
		log.Printf("Size Comparison for %s: ghost %d vs real %d", originFile, size, st.Size())
		ghostPdf, err3 := pdf.NewPdfReader(writerSeeker)
		if err3 != nil {
			log.Printf("Unable to open the buffer")
			return false
		}
		ghostPages, err := ghostPdf.GetNumPages()
		if err != nil {
			log.Printf("Unable to read number of pages from the buffer")
		}
		log.Printf("Buffer pages for %s: gosth %d vs real %d ", originFile, ghostPages, realPages)

		//need to set some margins on the size.
		//Origin is unkonwn, but there is sometimes small variations in the sizes
		//even if the file is not corrupted. It could come from a date recorded in the
		//file or images treatment
		if (size > (st.Size() - 200)) && (size < (st.Size() + 200)) {
			if ghostPages == realPages {
				return false
			}
		}
		return true

	} else if os.IsNotExist(err) {
		return true
	} else {
		// Schrodinger: file may or may not exist. See err for details.
		// Therefore, do *NOT* use !os.IsNotExist(err) to test for file existence
		log.Printf("File %s is not writeable or readable \n", originFile)
		return true
	}
	return true
}
