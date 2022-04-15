package urlchecker

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"bitbucket.org/waseka/waseka-xml-generator/utils"
)

var wg sync.WaitGroup
var mut sync.Mutex

type Rubrikk struct {
	XMLName xml.Name        `xml:"rubrikk"`
	Advert  []RubrikkAdvert `xml:"ad"`
}

var CategoryMap = map[string]string{
	"feed1.xml": "residential-for-sale",
	"feed2.xml": "residential-to-rent",
	"feed3.xml": "commercial-for-sale",
	"feed4.xml": "commercial-to-rent",
}

type RubrikkAdvert struct {
	XMLName              xml.Name `xml:"ad"`
	AdHeadline           string   `xml:"ad__headline"`
	Description          string   `xml:"ad__description"`
	Price                float32  `xml:"ad__price"`
	PriceCurrency        string   `xml:"ad__price_currency"`
	CompanyURL           string   `xml:"advertiser__company_homepage_url"`
	Mobile               string   `xml:"advertiser__mobile"`
	Phone                string   `xml:"advertiser__phone"`
	URL                  string   `xml:"ad__url"`
	Thumbnail            string   `xml:"ad__imageurl"`
	AdvertImages         []string `xml:"ad__all_imageurls>image"`
	MainCategoryOriginal string   `xml:"maincategory_original"`
	CategoryOriginal     string   `xml:"category_original"`
	MunicipalityCity     string   `xml:"location__municipality_city"`
	PostalName           string   `xml:"location__postal_name"`
	Postcode             string   `xml:"location__zip_postal_code"`
	Lat                  float32  `xml:"location__latitude"`
	Lng                  float32  `xml:"location__longitude"`
	StreetAddress        string   `xml:"location__streetaddress"`
}

func loadFeeds() []string {
	if _, err := os.Stat("feeds"); err != nil {
		err = os.MkdirAll("feeds", 0777)

		if err != nil {
			panic(err.Error())
		}
	}

	files, err := ioutil.ReadDir("feeds")
	if err != nil {
		panic(err.Error())
	}
	var feedList []string
	for _, file := range files {
		feedList = append(feedList, file.Name())
	}

	return feedList
}

func CheckURL() {
	// make empty url-error-log.txt before testing
	utils.EmptyFile("url-error-log.txt")
	fileList := loadFeeds()

	if len(fileList) <= 0 {
		log.Fatal("No file available to test on feeds directory")
	}

	for _, file := range fileList {
		writeLogTitle(file)
		xmlFile, err := os.Open("feeds/" + file)

		if err != nil {
			panic(err.Error())
		}

		fmt.Println("Successfully Opened " + file)

		defer xmlFile.Close()

		byteValue, _ := ioutil.ReadAll(xmlFile)

		var rubrikk Rubrikk

		xml.Unmarshal(byteValue, &rubrikk)

		for i := 0; i < len(rubrikk.Advert); i++ {
			// wg.Add(1)
			sendRequest(rubrikk.Advert[i].URL, i)
		}
		// wg.Wait()
	}
}

func sendRequest(url string, requestNumber int) {
	// defer wg.Done()
	http.DefaultClient.Timeout = time.Minute * 10
	res, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	if res.StatusCode != 200 {
		createLog(url, res.StatusCode)
	}

	fmt.Printf("[%d] [%d] %s\n", requestNumber, res.StatusCode, url)
}

func writeLogTitle(title string) {
	f, err := os.OpenFile("url-error-log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)

	if err != nil {
		panic(err.Error())
	}

	output := []byte("\n---------- " + strings.ToUpper(CategoryMap[title]) + " ----------")
	_, err = f.Write([]byte(append(output, "\n\n"...)))
	if err != nil {
		panic(err.Error())
	}

	f.Close()
}

func createLog(url string, statusCode int) {
	f, err := os.OpenFile("url-error-log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)

	if err != nil {
		panic(err.Error())
	}
	now := time.Now().Local().Format("2006-01-02 15:04:05")
	output := []byte("[" + now + "] [" + strconv.Itoa(statusCode) + "] - " + url)
	_, err = f.Write([]byte(append(output, "\n"...)))
	if err != nil {
		panic(err.Error())
	}

	f.Close()
}
