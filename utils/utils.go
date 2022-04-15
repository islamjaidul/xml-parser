package utils

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
)

var PropertyTableMap = map[string]string{
	"residential-for-sale": "residential_for_sales",
	"residential-to-rent":  "residential_to_rents",
	"commercial-for-sale":  "commercial_for_sales",
	"commercial-to-rent":   "commercial_to_rents",
}

var FileNameMap = map[string]string{
	"residential-for-sale": "feed1.xml",
	"residential-to-rent":  "feed2.xml",
	"commercial-for-sale":  "feed3.xml",
	"commercial-to-rent":   "feed4.xml",
}

type Property struct {
	Id               int
	AgentBranchId    int
	PropertyType     string
	Price            sql.NullFloat64
	PriceType        sql.NullString
	Postcode         string
	PostalName       string
	City             string
	StreetAddress    string
	ShortDescription string
	Lat              float32
	Lng              float32
	Bed              sql.NullInt32
	Bathroom         sql.NullInt32
	PropertyImages   string
	AdvertImages     []string
	Thumbnail        string
	IsSold           sql.NullInt32
	IsXmlParsed      int
	PublishedAt      sql.NullString
	ExpiredAt        sql.NullString
	ActiveAt         sql.NullString
	DeletedAt        sql.NullString
	IsSynced         int
	BranchId         int
	BranchName       string
	Mobile           sql.NullString
}

func VerifyInput(propertyCategory string) (string, error) {
	availableInput := []string{
		"residential-for-sale",
		"residential-to-rent",
		"commercial-for-sale",
		"commercial-to-rent",
	}

	for i := 0; i < len(availableInput); i++ {
		if availableInput[i] == propertyCategory {
			return propertyCategory, nil
		}
	}

	return "", fmt.Errorf("Invalid input, input should contains only - " + strings.Join(availableInput, ", "))
}

func VerifyExecutionType(executionType string) (string, error) {
	availableInput := []string{
		"parse",
		"test",
	}

	for i := 0; i < len(availableInput); i++ {
		if availableInput[i] == executionType {
			return executionType, nil
		}
	}

	return "", fmt.Errorf("Invalid input, input should contains only - " + strings.Join(availableInput, ", "))
}

func SaleOrLet(propertyCategory string) string {
	saleOrLet := strings.Split(propertyCategory, "-")
	isSaleOrLet := strings.Join(saleOrLet[1:], " ")

	if isSaleOrLet == "to rent" {
		return "to let"
	}
	return "for sale"
}

func PropertyTitle(property Property, propertyCategory string) string {
	if propertyCategory == "residential-for-sale" || propertyCategory == "residential-to-rent" {
		bed := strconv.Itoa(int(property.Bed.Int32))
		if bed == "0" {
			bed = "Studio"
		}
		var title string

		if strings.ToLower(property.PropertyType) != "land" {
			title = bed + " bedroom"
			if strings.ToLower(property.PropertyType) == "other" {
				return title + " property " + SaleOrLet(propertyCategory)
			}
			return title + " " + strings.ToLower(property.PropertyType) + " " + SaleOrLet(propertyCategory)
		}

		return strings.Title(property.PropertyType) + " " + SaleOrLet(propertyCategory)
	}

	// Only for commercial property category
	if strings.ToLower(property.PropertyType) == "other" {
		return "Commercial property " + SaleOrLet(propertyCategory)
	}
	return strings.Title(property.PropertyType) + " " + SaleOrLet(propertyCategory)
}

func PropertyURL(propertyCategory string, propertyId int) string {
	return os.Getenv("APP_URL") + "/single-property/" + propertyCategory + "/" + strconv.Itoa(propertyId)
}

func CompanyURL(branchName string, branchId int) string {
	re, err := regexp.Compile("[^a-zA-Z0-9]+")

	if err != nil {
		log.Fatal(err)
	}

	branchName = strings.ToLower(branchName)
	branchName = re.ReplaceAllString(branchName, " ")
	branchName = strings.Join(strings.Split(branchName, " "), "-")

	return os.Getenv("APP_URL") + "/agent/search/company/profile/" + branchName + "-" + strconv.Itoa(branchId)
}

func RemoveExistentContents(dirName string) {
	os.RemoveAll(dirName)
	os.MkdirAll(dirName, 0777)
}

func EmptyFile(filePath string) {
	if _, err := os.Stat(filePath); err != nil {
		err := os.WriteFile(filePath, []byte(""), 0755)
		if err != nil {
			fmt.Printf("Unable to write file: %v", err)
		}
	}

	if err := os.Truncate(filePath, 0); err != nil {
		panic(err.Error())
	}
}

func TransferFeeds() {
	createPublicXmlFile()

	exportPath := os.Getenv("EXPORT_PATH") + "/feeds"
	os.RemoveAll(exportPath)
	err := os.Rename("feeds", exportPath)

	if err != nil {
		panic(err.Error())
	}

	err = os.Chmod(exportPath, 0777)

	if err != nil {
		panic(err.Error())
	}

	err = os.MkdirAll("feeds", 0777)

	if err != nil {
		panic(err.Error())
	}
}

func createPublicXmlFile() {
	type FeedXml struct {
		XMLName  xml.Name `xml:"links"`
		Location []string `xml:"loc"`
	}

	// Reading all parsed xml file from feeds directory to generate feed.xml file
	files, err := ioutil.ReadDir("feeds")
	if err != nil {
		panic(err.Error())
	}

	// Removing feed.xml file if previously exist
	_, err = os.Stat("feed.xml")
	if err == nil {
		err = os.Remove("feed.xml")
		if err != nil {
			panic(err.Error())
		}
	}

	var feedXml FeedXml
	for _, file := range files {
		feedXml.Location = append(feedXml.Location, os.Getenv("APP_URL")+"/feeds/"+file.Name())
	}

	// Creating feed.xml file
	if len(feedXml.Location) > 0 {
		output, _ := xml.MarshalIndent(feedXml, "", "    ")
		output = []byte(xml.Header + string(output))
		f, err := os.OpenFile("feed.xml", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			panic(err.Error())
		}

		_, err = f.Write([]byte(append(output, "\n"...)))
		if err != nil {
			panic(err.Error())
		}

		f.Close()

		// Moving feed.xml file to export path
		err = os.Rename("feed.xml", os.Getenv("EXPORT_PATH")+"/feed.xml")

		if err != nil {
			panic(err.Error())
		}

		// Giving feed.xml file permission
		err = os.Chmod(os.Getenv("EXPORT_PATH")+"/feed.xml", 0777)

		if err != nil {
			panic(err.Error())
		}
	}
}

func PriceInDecimal(price float64) decimal.Decimal {
	decimalPrice, err := decimal.NewFromString(fmt.Sprint(price))
	if err != nil {
		panic(err)
	}

	return decimalPrice
}
