//Deployed with pwd changes
// CHECKED IN
//Added accountid to response obj
//Added fixes to Display Product Segment
//Added Acounttypes + Coffee Types
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

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

type AccountDetails struct {
	View          bool   `json:"viewaccount"`
	Update        bool   `json:"updateaccount"`
	AccountID     string `json:"accountid"`
	AccountOwner  string `json:"accountowner"`
	AccountName   string `json:"accountname"`
	Accounttypeid string `json:"accounttypeid"`
	Aliases       string `json:"aliases"`
	Phone         string `json:"phone"`
	Email         string `json:"email"`
	// Fax 					   string 				  `json:"fax"`
	Website            string         `json:"website"`
	ApproxAnnualRev    string         `json:"approxannualrev"`
	Productsegmentid   string         `json:"productsegmentid"`
	Shipping_Continent string         `json:"shippingtocontinent"`
	Shipping_Country   string         `json:"shippingtocountry"`
	OtherInformation   string         `json:"otherinfo"`
	BillingInfo        []BillingInfo  `json:"billinginformation"`
	ShippingInfo       []ShippingInfo `json:"shippinginformation"`

	Productsegment     []ProductSegments     `json:"Productsegment"`
	CoffeeTypes        []CoffeeTypes         `json:"coffeetypes"`
	AccountTypes       []AccountsInformation `json:"accounttypes"`
	ProfileSectionInfo []ProfileSectionInfo  `json:"profilesectioninfo"`
	B_Street           string                `json:"billing_street"`
	B_City             string                `json:"billing_city"`
	B_State            string                `json:"billing_state"`
	B_PostalCode       string                `json:"billing_postalcode"`
	B_Country          string                `json:"billing_country"`
	S_Street           string                `json:"shipping_street"`
	S_City             string                `json:"shipping_city"`
	S_State            string                `json:"shipping_state"`
	S_PostalCode       string                `json:"shipping_postalcode"`
	S_Country          string                `json:"shipping_country"`
	Manfacunit         bool                   `json:"manfacunit"`
	Instcoffee         bool                   `json:"instcoffee"`
	SampleReady        bool                   `json:"sample_ready"`
	Coffeetypeid       string                `json:"coffeetypeid"`
}
type ProductSegments struct {
	Productsegmentid string `json:"id"`
	Productsegment   string `json:"productsegment"`
}
type AccountsInformation struct {
	Accounttypeid string `json:"id"`
	Accounttype   string `json:"accounttype"`
}
type CoffeeTypes struct {
	CoffeeType   string `json:"coffeetype"`
	CoffeeTypeId string `json:"coffeetypeid"`
}
type BillingInfo struct {
	B_BillingID  string `json:"billing_id"`
	B_Street     string `json:"billing_street"`
	B_City       string `json:"billing_city"`
	B_State      string `json:"billing_state"`
	B_PostalCode string `json:"billing_postalcode"`
	B_Country    string `json:"billing_country"`
}
type ShippingInfo struct {
	S_ShippingID string `json:"shipping_id"`
	S_Street     string `json:"shipping_street"`
	S_City       string `json:"shipping_city"`
	S_State      string `json:"shipping_state"`
	S_PostalCode string `json:"shipping_postalcode"`
	S_Country    string `json:"shipping_country"`
}
type ProfileSectionInfo struct {
	Manfacunit   bool    `json:"manfacunit"`
	Instcoffee   bool    `json:"instcoffee"`
	SampleReady        bool    `json:"sample_ready"`
	Coffeetypeid string `json:"coffeetypeid"`
}

func getAccountHyperlink(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var account AccountDetails
	err := json.Unmarshal([]byte(request.Body), &account)
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
	var rows *sql.Rows
	var ad AccountDetails
	var productSegment ProductSegments
	var accInfo AccountsInformation
	var psi ProfileSectionInfo
	var coffeeType CoffeeTypes
	var bi BillingInfo
	var si ShippingInfo

	if account.View {
		var accowner, accname, aliases, phone, email, website, otherinfo, shipcountry, shipcont, approxannualrev sql.NullString
		fmt.Println("Entered View Account Module")
		sqlStatementav1 := `SELECT
							accountid,accountname,accounttypeid,
							aliases,phone,email,approxannualrev,
							website,productsegmentid,otherinformation,shipping_country,
							shipping_continent,account_owner
							FROM
							dbo.accounts_master
							where accountid=$1`
		rows, err = db.Query(sqlStatementav1, account.AccountID)
		if err != nil {
			log.Println(err)
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}
		defer rows.Close()
		for rows.Next() {

			err = rows.Scan(&ad.AccountID, &accname, &ad.Accounttypeid,
				&aliases, &phone, &email, &approxannualrev,
				&website, &ad.Productsegmentid, &otherinfo, &shipcountry,
				&shipcont, &accowner)

		}
		ad.AccountOwner = accowner.String
		ad.AccountName = accname.String
		ad.Aliases = aliases.String
		ad.Phone = phone.String
		ad.Email = email.String
		ad.Website = website.String
		ad.OtherInformation = otherinfo.String
		ad.Shipping_Country = shipcountry.String
		ad.Shipping_Continent = shipcont.String
		ad.ApproxAnnualRev = approxannualrev.String
		if ad.Productsegmentid != "" {
			s := strings.Split(ad.Productsegmentid, ",")
			for i, s := range s {
				log.Println("get product segments", i, s)
				sqlStatement := `SELECT productsegmentid, productsegment FROM dbo.cms_account_product_segment_master where productsegmentid=$1`
				rows, err = db.Query(sqlStatement, s)

				if err != nil {
					log.Println(err)
					log.Println("unable to get product segment", ad.Productsegment)
				}

				defer rows.Close()
				for rows.Next() {

					err = rows.Scan(&productSegment.Productsegmentid, &productSegment.Productsegment)
					allProductSegments := append(ad.Productsegment, productSegment)
					ad.Productsegment = allProductSegments
					log.Println("added one product segement", allProductSegments)
				}
			}

		}
		if ad.Accounttypeid != "" {
			z := strings.Split(ad.Accounttypeid, ",")
			for i, z := range z {
				log.Println("get accounts", i, z)
				sqlStatement := `SELECT accounttypeid, accounttype FROM dbo.cms_account_type_master where accounttypeid=$1`
				rows, err = db.Query(sqlStatement, z)

				if err != nil {
					log.Println(err)
					log.Println("unable to add account type")
				}

				defer rows.Close()
				for rows.Next() {

					err = rows.Scan(&accInfo.Accounttypeid, &accInfo.Accounttype)
					allAccounts := append(ad.AccountTypes, accInfo)
					ad.AccountTypes = allAccounts
					log.Println("added one account", allAccounts)
				}
			}
		}
		// Get Coffee Types & other Lead Info

		sqlStatementli1 := `SELECT manfacunit,instcoffee,sampleready,coffeetypeid
							from
							dbo.cms_leads_master 
							where accountid=$1`
		rows, err = db.Query(sqlStatementli1, account.AccountID)
		if err != nil {
			log.Println(err)
			log.Println("unable to find shipping address for the account")
		}
		defer rows.Close()

		for rows.Next() {

			err = rows.Scan(&psi.Manfacunit, &psi.Instcoffee, &psi.SampleReady, &psi.Coffeetypeid)
			profileInfo := append(ad.ProfileSectionInfo, psi)
			ad.ProfileSectionInfo = profileInfo
			log.Println("added shipping info")
		}

		//Billing Info

		log.Println("get Billing Info")
		sqlStatementbi1 := `SELECT billingid,street,city,stateprovince,postalcode,country
							from dbo.accounts_billing_address_master
							where accountid=$1`

		rows, err = db.Query(sqlStatementbi1, account.AccountID)

		if err != nil {
			log.Println(err)
			log.Println("unable to find billing address for the account")
		}

		defer rows.Close()
		for rows.Next() {

			err = rows.Scan(&bi.B_BillingID, &bi.B_Street, &bi.B_City, &bi.B_State, &bi.B_PostalCode, &bi.B_Country)
			billInfo := append(ad.BillingInfo, bi)
			ad.BillingInfo = billInfo
			log.Println("added billing info")
		}

		//Get Shipping Info

		log.Println("get Shipping Info")
		sqlStatementsi1 := `SELECT shippingid,street,city,stateprovince,postalcode,country
							from dbo.accounts_shipping_address_master
							where accountid=$1`

		rows, err = db.Query(sqlStatementsi1, account.AccountID)

		if err != nil {
			log.Println(err)
			log.Println("unable to find shipping address for the account")
		}
		defer rows.Close()
		for rows.Next() {

			err = rows.Scan(&si.S_ShippingID, &si.S_Street, &si.S_City, &si.S_State, &si.S_PostalCode, &si.S_Country)
			shipInfo := append(ad.ShippingInfo, si)
			ad.ShippingInfo = shipInfo
			log.Println("added shipping info")
		}
		if psi.Coffeetypeid != "" {
			w := strings.Split(psi.Coffeetypeid, ",")
			for i, w := range w {
				log.Println("get coffeetypes", i, w)
				sqlStatement := `SELECT id, coffeetype FROM dbo.cms_coffeetype_master where id=$1`

				rows, err = db.Query(sqlStatement, w)

				if err != nil {
					log.Println(err)
					log.Println("unable to add coffee type")
				}

				defer rows.Close()
				for rows.Next() {
					err = rows.Scan(&coffeeType.CoffeeTypeId, &coffeeType.CoffeeType)
					allCoffeeTypes := append(ad.CoffeeTypes, coffeeType)
					ad.CoffeeTypes = allCoffeeTypes
					log.Println("added one coffeetype", allCoffeeTypes)
				}
			}
		}

		res, _ := json.Marshal(ad)
		return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil

	} else if account.Update {

		sqlStatementau1 := `UPDATE dbo.accounts_master SET 
		accountname=$1,
		accounttypeid=$2,
		aliases=$3,
		phone=$4,
		email=$5,
		website=$6,
		approxannualrev=$7,
		productsegmentid=$8,
		otherinformation=$9,
		shipping_continent=$10,
		shipping_country=$11
		where
		accountid=$12`

		rows, err = db.Query(sqlStatementau1,
			account.AccountName,
			account.Accounttypeid,
			account.Aliases,
			account.Phone,
			account.Email,
			account.Website,
			account.ApproxAnnualRev,
			account.Productsegmentid,
			account.OtherInformation,
			account.Shipping_Continent,
			account.Shipping_Country,
			account.AccountID)

		//Update Profile section in Leads table
		sqlStatementapu1 := `UPDATE dbo.cms_leads_master SET 
							manfacunit=$1,
							instcoffee=$2,
							sampleready=$3,
							coffeetypeid=$4
							where
							accountid=$5`

		rows, err = db.Query(sqlStatementapu1,
			account.Manfacunit,
			account.Instcoffee,
			account.SampleReady,
			account.Coffeetypeid,
			account.AccountID)
		log.Println("Profile Section of corresponding Lead is updated in leads table")

		// Inserting Address Details
		log.Println(bi.B_Street, bi.B_City, bi.B_State, bi.B_PostalCode, bi.B_Country)
		sqlStatementb1 := `UPDATE dbo.accounts_billing_address_master SET street=$1, city=$2, stateprovince=$3, postalcode=$4, country=$5 where accountid=$6`

		rows, err = db.Query(sqlStatementb1, account.B_Street, account.B_City, account.B_State,
			account.B_PostalCode, account.B_Country, account.AccountID)

		sqlStatements1 := `UPDATE dbo.accounts_shipping_address_master SET street=$1, city=$2, stateprovince=$3, postalcode=$4, country=$5 where accountid=$6`

		rows, err = db.Query(sqlStatements1, account.S_Street, account.S_City, account.S_State,
			account.S_PostalCode, account.S_Country, account.AccountID)
		log.Println(si.S_Street, si.S_City, si.S_State, si.S_PostalCode, si.S_Country)
		if err != nil {
			log.Println("Update to Account table failed")
			log.Println(err.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

		res, _ := json.Marshal("Success")
		log.Println("Update to Account table Success")
		return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil

	}
	return events.APIGatewayProxyResponse{200, headers, nil, string("Success"), false}, nil
}

func main() {
	lambda.Start(getAccountHyperlink)
}
