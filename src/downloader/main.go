package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/gocolly/colly/v2"
	"github.com/sirupsen/logrus"
)

const (
	nfzURL       = "https://www.nfz.gov.pl/aktualnosci/aktualnosci-centrali/wykazy-placowek-udzielajacych-swiadczen-w-zwiazku-z-przeciwdzialaniem-rozprzestrzenianiu-koronawirusa,7624.html?fbclid=IwAR0yR7WJqZDNcHN_a0vnsxyGBtWNbKm3cyrI9rxQHz4CoaH6wG17thG7wPc"
	pdfPrefix    = "https://www.nfz.gov.pl"
	S3BucketName = "coronavirus.institutions.data"
)

func main() {
	logger := logrus.New()
	log := logger.WithField("service", "downloader")
	if _, validRegion := os.LookupEnv("AWS_REGION"); !validRegion {
		log.Panic("AWS_REGION variable must be set")
	}

	// s3 session
	sess := session.Must(session.NewSession())
	uploader := s3manager.NewUploader(sess)

	// scraping collector
	collector := colly.NewCollector()

	collector.OnHTML("div[class='news-module']", func(element *colly.HTMLElement) {
		element.ForEach("a[class='ckeditor-style-5']", func(i int, element *colly.HTMLElement) {
			provinceName := element.Text
			provinceURL := element.Attr("href")

			if provinceName == "" || provinceURL == "" {
				log.Error("failed to acquire province data")
			} else {
				pLog := log.WithField("province", provinceName)
				pCollector := colly.NewCollector()

				provinceCounter := 0
				pCollector.OnHTML("a[class='ckeditor-style-4']", func(element *colly.HTMLElement) {
					if url := element.Attr("href"); url != "" {
						pdfName := fmt.Sprintf("%s_%d.pdf", provinceName, provinceCounter)
						if err := DownloadToS3(pdfName, fmt.Sprintf("%s/%s", pdfPrefix, url), uploader); err != nil {
							pLog.WithError(err).Error("failed to Download file")
						} else {
							pLog.Infof("%s downloaded", pdfName)
						}
					} else {
						pLog.Error("failed to acquire pdf url")
					}
					provinceCounter += 1
				})

				if err := pCollector.Visit(provinceURL); err != nil {
					pLog.WithError(err).Error("failed to enter province url")
				}
			}
		})
	})

	if err := collector.Visit(nfzURL); err != nil {
		log.WithError(err).Fatalf("failed to visit NFZ site")
	}
}

// DownloadToS3 will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func DownloadToS3(filepath string, url string, uploader *s3manager.Uploader) error {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(S3BucketName),
		Key:    aws.String(filepath),
		Body:   resp.Body,
	})
	if err != nil {
		return fmt.Errorf("failed to upload to S3, %w", err)
	}
	fmt.Printf("file uploaded to %s\n", result.Location)
	return nil
}
