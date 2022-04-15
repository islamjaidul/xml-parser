package parser

import (
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"bitbucket.org/waseka/waseka-xml-generator/utils"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"github.com/shopspring/decimal"
)

const LIMIT = 1000

var totalNumberPropertyParsed int
var category string
var allXMLParsedPropertyIds []string

var wg sync.WaitGroup
var mut sync.Mutex

type RubrikkAdvert struct {
	XMLName              xml.Name        `xml:"ad"`
	Id                   int             `xml:"ad__number_reference_id"`
	AdHeadline           string          `xml:"ad__headline"`
	Description          string          `xml:"ad__description"`
	Price                decimal.Decimal `xml:"ad__price"`
	PriceCurrency        string          `xml:"ad__price_currency"`
	CompanyURL           string          `xml:"advertiser__company_homepage_url"`
	Mobile               string          `xml:"advertiser__mobile"`
	Phone                string          `xml:"advertiser__phone"`
	URL                  string          `xml:"ad__url"`
	Thumbnail            string          `xml:"ad__imageurl"`
	AdvertImages         []string        `xml:"ad__all_imageurls>image"`
	MainCategoryOriginal string          `xml:"maincategory_original"`
	CategoryOriginal     string          `xml:"category_original"`
	MunicipalityCity     string          `xml:"location__municipality_city"`
	PostalName           string          `xml:"location__postal_name"`
	Postcode             string          `xml:"location__zip_postal_code"`
	Lat                  float32         `xml:"location__latitude"`
	Lng                  float32         `xml:"location__longitude"`
	StreetAddress        string          `xml:"location__streetaddress"`
	Bed                  int32           `xml:"real_estate__beds,omitempty"`
	Bathroom             int32           `xml:"real_estate__number_of_bathrooms,omitempty"`
}

func initialLoad(propertyCategory string) {
	// intial setup
	totalNumberPropertyParsed = 0
	category = propertyCategory
}

func totalRecords(db *sql.DB) int {
	today := time.Now().Local().Format("2006-01-02")
	query := fmt.Sprintf("SELECT count(*) from %s where published_at is not null and date(expired_at) > \"%s\" and active_at is not null and deleted_at is null and is_sold is null and is_synced = 1", utils.PropertyTableMap[category], today)
	rows, err := db.Query(query)

	if err != nil {
		panic(err.Error())
	}

	var count int

	for rows.Next() {
		if err := rows.Scan(&count); err != nil {
			panic(err.Error())
		}
	}

	return count
}

func ParseToXML(propertyCategory string) {
	// setup intial dependency
	initialLoad(propertyCategory)

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	db, err := sql.Open("mysql", os.Getenv("MYSQL_USER")+":"+os.Getenv("MYSQL_PASSWORD")+"@tcp("+os.Getenv("MYSQL_HOST")+":"+os.Getenv("MYSQL_PORT")+")/"+os.Getenv("MYSQL_DATABASE"))

	if err != nil {
		panic(err.Error())
	}

	total := int(math.Ceil(float64(totalRecords(db)) / LIMIT))
	db.Close()

	offset := 0
	for i := 0; i < total; i++ {
		wg.Add(1)
		go execute(offset)
		offset += LIMIT
	}

	wg.Wait()

	if len(allXMLParsedPropertyIds) > 0 {
		updateProperty()
	}

	if totalNumberPropertyParsed > 0 {
		updateXML("feeds/" + utils.FileNameMap[category])
		createLog()
	}
}

func createLog() {
	f, err := os.OpenFile("log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)

	if err != nil {
		panic(err.Error())
	}
	now := time.Now().Local().Format("2006-01-02 15:04:05")
	output := []byte("[" + now + "] - Total " + strings.ToUpper(category) + " properties parsed - " + strconv.Itoa(totalNumberPropertyParsed))
	_, err = f.Write([]byte(append(output, "\n"...)))
	if err != nil {
		panic(err.Error())
	}

	f.Close()
}

func getCityNameByPostcode(postcode string) string {
	db, err := sql.Open("mysql", os.Getenv("MYSQL_USER")+":"+os.Getenv("MYSQL_PASSWORD")+"@tcp("+os.Getenv("MYSQL_HOST")+":"+os.Getenv("MYSQL_PORT")+")/"+os.Getenv("MYSQL_DATABASE"))

	if err != nil {
		panic(err.Error())
	}

	defer db.Close()

	postcode = strings.ToLower(strings.ReplaceAll(postcode, " ", ""))
	// fmt.Println(postcode)

	query := fmt.Sprintf("SELECT place, searchable_keyword from geolytix_locations where searchable_keyword = \"%s\"", postcode)
	rows, err := db.Query(query)
	if err != nil {
		panic(err.Error())
	}

	var place string
	var keyword string

	for rows.Next() {
		if err := rows.Scan(&place, &keyword); err != nil {
			panic(err.Error())
		}
	}

	placeSlice := strings.Split(place, ", ")
	if len(placeSlice) > 1 {
		return placeSlice[1]
	}
	return ""
}

func execute(offset int) {
	db, err := sql.Open("mysql", os.Getenv("MYSQL_USER")+":"+os.Getenv("MYSQL_PASSWORD")+"@tcp("+os.Getenv("MYSQL_HOST")+":"+os.Getenv("MYSQL_PORT")+")/"+os.Getenv("MYSQL_DATABASE"))
	defer wg.Done()

	if err != nil {
		panic(err.Error())
	}

	today := time.Now().Local().Format("2006-01-02")
	query := fmt.Sprintf("SELECT ab.id as branch_id, ab.branch_name, ab.contact_phone, p.id, p.agent_branch_id, p.property_type, p.price, p.price_type, p.postcode, p.address_line1, p.short_description, p.city, p.lat, p.lng, p.bed, p.bathroom, p.property_images, p.thumbnail, p.is_sold, p.is_xml_parsed, p.published_at, p.expired_at, p.active_at, p.deleted_at, p.is_synced from %s as p, agent_branches as ab where p.agent_branch_id = ab.id and p.published_at is not null and date(p.expired_at) > \"%s\" and p.active_at is not null and p.deleted_at is null and p.is_sold is null and p.is_synced = 1 limit %s, %s", utils.PropertyTableMap[category], today, strconv.Itoa(offset), strconv.Itoa(LIMIT))
	results, err := db.Query(query)
	if err != nil {
		panic(err.Error())
	}
	db.Close()

	fmt.Println(offset)

	for results.Next() {
		var property utils.Property

		err = results.Scan(
			&property.BranchId,
			&property.BranchName,
			&property.Mobile,
			&property.Id,
			&property.AgentBranchId,
			&property.PropertyType,
			&property.Price,
			&property.PriceType,
			&property.Postcode,
			&property.StreetAddress,
			&property.ShortDescription,
			&property.City,
			&property.Lat,
			&property.Lng,
			&property.Bed,
			&property.Bathroom,
			&property.PropertyImages,
			&property.Thumbnail,
			&property.IsSold,
			&property.IsXmlParsed,
			&property.PublishedAt,
			&property.ExpiredAt,
			&property.ActiveAt,
			&property.DeletedAt,
			&property.IsSynced,
		)

		if err != nil {
			panic(err.Error())
		}

		// Setting property city as postalname
		property.PostalName = property.City

		// Get the city name from geolytix_locations table by another sql query
		// If not found then the previous city name will be remain
		if city := getCityNameByPostcode(property.Postcode); city != "" {
			property.City = city
		}

		// Residential bed setup
		if category == "residential-for-sale" || category == "residential-to-rent" {
			// If bed is 0 then bed should be 1 and bathroom should be 1
			if property.Bed.Valid && property.Bed.Int32 == 0 {
				property.Bed.Int32 = 1
				property.Bathroom.Int32 = 1
			}
		}
		// Commercial bed and bathroom ignore
		if category == "commercial-for-sale" || category == "commercial-to-rent" {
			property.Bed.Int32 = 0
			property.Bathroom.Int32 = 0
		}

		type Image struct {
			URL string
		}

		type ImageList struct {
			Gallery []Image
		}

		var imageList ImageList
		json.Unmarshal([]byte(property.PropertyImages), &imageList)

		for i := 0; i < len(imageList.Gallery); i++ {
			property.AdvertImages = append(property.AdvertImages, imageList.Gallery[i].URL)
		}

		if property.Price.Valid {
			mut.Lock()
			createXML(property)
			allXMLParsedPropertyIds = append(allXMLParsedPropertyIds, strconv.Itoa(property.Id))
			mut.Unlock()
		}
	}
}

func updateProperty() {
	db, err := sql.Open("mysql", os.Getenv("MYSQL_USER")+":"+os.Getenv("MYSQL_PASSWORD")+"@tcp("+os.Getenv("MYSQL_HOST")+":"+os.Getenv("MYSQL_PORT")+")/"+os.Getenv("MYSQL_DATABASE"))

	if err != nil {
		panic(err.Error())
	}

	_, err = db.Exec("UPDATE " + utils.PropertyTableMap[category] + " SET is_xml_parsed = 1 WHERE id in (" + strings.Join(allXMLParsedPropertyIds, ", ") + ")")
	if err != nil {
		panic(err.Error())
	}
}

func updateXML(filePath string) {
	_, err := exec.Command("/bin/sh", "bash.sh", filePath).Output()
	if err != nil {
		panic(err.Error())
	}
}

func createXML(property utils.Property) {
	rubrikkAdvert := RubrikkAdvert{
		Id:                   property.Id,
		CompanyURL:           utils.CompanyURL(property.BranchName, property.BranchId),
		Mobile:               property.Mobile.String,
		Phone:                property.Mobile.String,
		AdHeadline:           utils.PropertyTitle(property, category),
		Description:          property.ShortDescription,
		Price:                utils.PriceInDecimal(property.Price.Float64),
		PriceCurrency:        "GBP",
		URL:                  utils.PropertyURL(category, property.Id),
		Thumbnail:            property.Thumbnail,
		MunicipalityCity:     property.City,
		PostalName:           property.PostalName,
		Postcode:             property.Postcode,
		StreetAddress:        property.StreetAddress,
		Lat:                  property.Lat,
		Lng:                  property.Lng,
		MainCategoryOriginal: category,
		CategoryOriginal:     utils.SaleOrLet(category),
		AdvertImages:         property.AdvertImages,
		Bed:                  property.Bed.Int32,
		Bathroom:             property.Bathroom.Int32,
	}
	output, _ := xml.MarshalIndent(rubrikkAdvert, "   ", "    ")
	fileName := utils.FileNameMap[category]
	f, err := os.OpenFile("feeds/"+fileName+"", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0777)

	if err != nil {
		panic(err.Error())
	}

	_, err = f.Write([]byte(append(output, "\n"...)))
	if err != nil {
		panic(err.Error())
	}

	f.Close()

	totalNumberPropertyParsed++
}
