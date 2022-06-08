// Deployed with password change
//Deployed with salutation scan code change
//Deployed with fixing countries code
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	_ "github.com/lib/pq"
)

const (
	host     = "ccl-psql-dev.cclxlbtddgmn.ap-south-1.rds.amazonaws.com"
	port     = 5432
	user     = "postgres"
	password = "Ccl_RDS_DB#2022"
	dbname   = "ccldevdb"
)

type AccountsInformation struct {
	Accounttypeid string `json:"id"`
	Accounttype   string `json:"accounttype"`
}

type ProductSegments struct {
	Productsegmentid int    `json:"id"`
	Productsegment   string `json:"productsegment"`
}

type PhoneCodes struct {
	Countryname string `json:"countryname"`
	Dialcode    string `json:"dialcode"`
}

type Salutations struct {
	Salutationid string `json:"id"`
	Salutation   string `json:"salutation"`
	Isactive     bool `json:"isactive"`
}

type Countries struct {
	Countryname string `json:"countryname"`
}

type CoffeeTypes struct {
	CoffeeTypeId string `json:"id"`
	CoffeeType   string `json:"coffeetype"`
}

type Input struct {
	Type          string `json:"type"`
	ContinentName string `json:"continentname"`
}

func getLeadCreationInfo(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var input Input
	err := json.Unmarshal([]byte(request.Body), &input)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}
	defer db.Close()

	// check db
	err = db.Ping()

	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}

	fmt.Println("Connected!")

	var res []byte

	var rows *sql.Rows
	if input.Type == "accountDetails" {
		log.Println("get accounts", input.Type)
		sqlStatement := `SELECT accounttypeid,accounttype FROM dbo.cms_account_type_master`
		rows, err = db.Query(sqlStatement)

		var allAccounts []AccountsInformation
		defer rows.Close()
		for rows.Next() {
			var account AccountsInformation
			err = rows.Scan(&account.Accounttypeid, &account.Accounttype)
			allAccounts = append(allAccounts, account)
		}

		res, _ = json.Marshal(allAccounts)

	} else if input.Type == "productsegments" {
		log.Println("get product segments", input.Type)
		sqlStatement := `SELECT productsegmentid,productsegment FROM dbo.cms_account_product_segment_master`
		rows, err = db.Query(sqlStatement)

		var allProductSegments []ProductSegments
		defer rows.Close()
		for rows.Next() {
			var productSegment ProductSegments
			err = rows.Scan(&productSegment.Productsegmentid, &productSegment.Productsegment)
			allProductSegments = append(allProductSegments, productSegment)
		}

		res, _ = json.Marshal(allProductSegments)
	} else if input.Type == "phonecodes" {
		log.Println("get phone codes", input.Type)
		sqlStatement := `SELECT "Country_Name", "Dial" FROM dbo.cms_phonecodes_master`
		rows, err = db.Query(sqlStatement)

		var allPhoneCodes []PhoneCodes
		defer rows.Close()
		for rows.Next() {
			var phoneCode PhoneCodes
			err = rows.Scan(&phoneCode.Countryname, &phoneCode.Dialcode)
			allPhoneCodes = append(allPhoneCodes, phoneCode)
		}

		res, _ = json.Marshal(allPhoneCodes)
	} else if input.Type == "countries" {
		log.Println("get countries", input.Type)
		sqlStatement := `select countryname from dbo.continents_countries_master where continentname=$1`

		// SELECT c.countryname from continents con inner join countries_master c on con.countryid=c.countryid where con.continent_name=$1`

		rows, err = db.Query(sqlStatement, input.ContinentName)

		var allCountries []Countries
		defer rows.Close()
		for rows.Next() {
			var country Countries
			err = rows.Scan(&country.Countryname)
			allCountries = append(allCountries, country)
		}

		res, _ = json.Marshal(allCountries)
	} else if input.Type == "coffeetypes" {
		log.Println("get coffeetypes", input.Type)
		sqlStatement := `SELECT id, coffeetype FROM dbo.cms_coffeetype_master`

		rows, err = db.Query(sqlStatement)

		var allCoffeeTypes []CoffeeTypes
		defer rows.Close()
		for rows.Next() {
			var coffeeType CoffeeTypes
			err = rows.Scan(&coffeeType.CoffeeTypeId, &coffeeType.CoffeeType)
			allCoffeeTypes = append(allCoffeeTypes, coffeeType)
		}

		res, _ = json.Marshal(allCoffeeTypes)
	} else if input.Type == "salutations" {
		log.Println("get salutations", input.Type)
		sqlStatement := `SELECT "salutationid", "salutation" FROM dbo.cms_salutation_master where isactive=$1`

		rows, err = db.Query(sqlStatement, 1)

		var allSalutations []Salutations
		defer rows.Close()
		for rows.Next() {
			var salutation Salutations
			err = rows.Scan(&salutation.Salutationid, &salutation.Salutation)
			allSalutations = append(allSalutations, salutation)
		}

		res, _ = json.Marshal(allSalutations)
	}

	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}

	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(getLeadCreationInfo)
}
