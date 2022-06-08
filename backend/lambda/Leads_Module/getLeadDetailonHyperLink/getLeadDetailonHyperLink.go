//Updated and deployed with pwd changes
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

type LeadDetails struct {
	Accountname                string                `json:"accountname"`
	Aliases                    string                `json:"aliases"`
	Accounttypeid              string                `json:"accounttypeid"`
	Website                    string                `json:"website"`
	Approximativeannualrevenue string                `json:"approxannualrev"`
	Productsegmentid           string                `json:"productsegmentid"`
	ContactSalutationid        int                   `json:"contact_saluatationid"`
	Salutations                Salutations           `json:"salutation"`
	Contactfirstname           string                `json:"contact_firstname"`
	Contactlastname            string                `json:"contact_lastname"`
	ContactPosition            string                `json:"contact_position"`
	ContactEmail               string                `json:"email"`
	ContactPhone               string                `json:"contact_phone"`
	ContactExtId               string                `json:"contact_ext"`
	ContactMobile              string                `json:"contact_mobile"`
	Manfacunit                 bool                   `json:"manfacunit"`
	Instcoffee                 bool                   `json:"instcoffee"`
	SampleReady                      bool                   `json:"sample_ready"`
	Coffeetypeid               string                `json:"coffeetypeid"`
	Productsegment             []ProductSegments     `json:"Productsegment"`
	CoffeeTypes                []CoffeeTypes         `json:"coffeetypes"`
	AccountTypes               []AccountsInformation `json:"accounttypes"`
	OtherInformation           string                `json:"otherinformation"`
	BillingStreetAddress       string                `json:"billing_street"`
	BillingCity                string                `json:"billing_citycode"`
	BillingState               string                `json:"billing_statecode"`
	BillingPostalCode          string                `json:"billing_postalcode"`
	BillingCountry             string                `json:"billing_countrycode"`
	ContactStreetAddress       string                `json:"contact_street"`
	ContactCity                string                `json:"contact_citycode"`
	ContactState               string                `json:"contact_statecode"`
	ContactPostalCode          string                `json:"contact_postalcode"`
	ContactCountry             string                `json:"contact_countrycode"`
	ShippingContinent          string                `json:"shipping_continent"`
	ShippingCountry            string                `json:"shipping_country"`
	Status                     string                `json:"status"`
	Leadscore                  int                   `json:"leadscore"`
	AuditLogDetails            []AuditLogGCPO        `json:"audit_log_crm_leads"`
}

type LeadId struct {
	Id string `json:"leadid"`
}

type Salutations struct {
	Salutationid string `json:"id"`
	Salutation   string `json:"salutation"`
}

type AccountsInformation struct {
	Accounttypeid string `json:"id"`
	Accounttype   string `json:"accounttype"`
}

type CoffeeTypes struct {
	CoffeeType   string `json:"coffeetype"`
	CoffeeTypeId string `json:"coffeetypeid"`
}

type ProductSegments struct {
	Productsegmentid int    `json:"id"`
	Productsegment   string `json:"productsegment"`
}
type AuditLogGCPO struct {
	CreatedDate    string `json:"createddate"`
	CreatedUserid  string `json:"createduserid"`
	ModifiedDate   string `json:"modifieddate"`
	ModifiedUserid string `json:"modifieduserid"`
	Description    string `json:"description"`
}

func getLeadDetailonHyperLink(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var id LeadId
	err := json.Unmarshal([]byte(request.Body), &id)
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
	var accountname,
		accounttypeid,
		phone,
		email,
		approxannualrev,
		website,
		contactfirstname,
		contactlastname,
		contact_position,
		contact_mobile,
		shipping_continent,
		shipping_country,
		coffeetypeid,
		aliases,
		otherinformation,
		masterstatus, contact_ext_id sql.NullString
	sqlStatement := `select 
			accountname,
			accounttypeid,
			phone,
			email,
			approxannualrev,
			website,
			productsegmentid,
			contactfirstname,
			contactlastname,
			manfacunit,
			instcoffee,
			sampleready,
			contact_salutationid,
			contact_position,
			contact_mobile,
			shipping_continent,
			shipping_country,
			coffeetypeid,
			aliases,
			otherinformation,
			masterstatus,
			leadscore,
			contact_ext_id
			from dbo.cms_leads_master where leadid=$1`

	rows, err = db.Query(sqlStatement, id.Id)

	if err != nil {
		log.Println(err)
		log.Println("unable to fetch the lead details")
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}

	var ld LeadDetails
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&accountname,
			&accounttypeid,
			&phone,
			&email,
			&approxannualrev,
			&website,
			&ld.Productsegmentid,
			&contactfirstname,
			&contactlastname,
			&ld.Manfacunit,
			&ld.Instcoffee,
			&ld.SampleReady,
			&ld.ContactSalutationid,
			&contact_position,
			&contact_mobile,
			&shipping_continent,
			&shipping_country,
			&coffeetypeid,
			&aliases,
			&otherinformation,
			&masterstatus,
			&ld.Leadscore,
			&contact_ext_id)

	}
	ld.Accountname = accountname.String
	ld.Accounttypeid = accounttypeid.String
	ld.ContactPhone = phone.String
	ld.ContactEmail = email.String
	ld.Approximativeannualrevenue = approxannualrev.String
	ld.Website = website.String
	// ld.Productsegmentid=productsegmentid.String
	ld.Contactfirstname = contactfirstname.String
	ld.Contactlastname = contactlastname.String
	ld.ContactPosition = contact_position.String
	ld.ContactMobile = contact_mobile.String
	ld.ShippingContinent = shipping_continent.String
	ld.ShippingCountry = shipping_country.String
	ld.Coffeetypeid = coffeetypeid.String
	ld.Aliases = aliases.String
	ld.OtherInformation = otherinformation.String
	ld.Status = masterstatus.String
	ld.ContactExtId = contact_ext_id.String

	log.Println("get salutations")
	sqlStatement = `SELECT salutationid, salutation FROM dbo.cms_salutation_master where salutationid=$1`

	rows, err = db.Query(sqlStatement, ld.ContactSalutationid)

	if err != nil {
		log.Println(err)
		log.Println("unable to add saluatation", ld.ContactSalutationid)
	}

	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&ld.Salutations.Salutationid, &ld.Salutations.Salutation)
		log.Println("added saluatation", &ld.Salutations.Salutationid)
	}

	sqlStatement = `SELECT street, city, stateprovince, postalcode, country FROM dbo.cms_leads_billing_address_master where leadid=$1`

	rows, err = db.Query(sqlStatement, id.Id)

	if err != nil {
		log.Println(err)
		log.Println("unable to add billing address", ld.ContactSalutationid)
	}

	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&ld.BillingStreetAddress, &ld.BillingCity, &ld.BillingState, &ld.BillingPostalCode, &ld.BillingCountry)
		log.Println("added billing address")
	}

	sqlStatement = `SELECT street, city, stateprovince, postalcode, country FROM dbo.cms_leads_shipping_address_master where leadid=$1`

	rows, err = db.Query(sqlStatement, id.Id)

	if err != nil {
		log.Println(err)
		log.Println("unable to add contact address", ld.ContactSalutationid)
	}

	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&ld.ContactStreetAddress, &ld.ContactCity, &ld.ContactState, &ld.ContactPostalCode, &ld.ContactCountry)
		log.Println("added contact address")
	}

	if ld.Productsegmentid != "" {
		s := strings.Split(ld.Productsegmentid, ",")
		for i, s := range s {
			log.Println("get product segments", i, s)
			sqlStatement := `SELECT productsegmentid, productsegment FROM dbo.cms_account_product_segment_master where productsegmentid=$1`
			rows, err = db.Query(sqlStatement, s)

			if err != nil {
				log.Println(err)
				log.Println("unable to add product segment", ld.ContactSalutationid)
			}

			defer rows.Close()
			for rows.Next() {
				var productSegment ProductSegments
				err = rows.Scan(&productSegment.Productsegmentid, &productSegment.Productsegment)
				allProductSegments := append(ld.Productsegment, productSegment)
				ld.Productsegment = allProductSegments
				log.Println("added one product segement", allProductSegments)
			}
		}

	}

	if ld.Coffeetypeid != "" {
		w := strings.Split(ld.Coffeetypeid, ",")
		for i, w := range w {
			log.Println("get coffeetypes", i, w)
			sqlStatement := `SELECT id, coffeetype FROM dbo.cms_coffeetype_master where id=$1`

			rows, err = db.Query(sqlStatement, w)

			if err != nil {
				log.Println(err)
				log.Println("unable to add coffee type", ld.ContactSalutationid)
			}

			defer rows.Close()
			for rows.Next() {
				var coffeeType CoffeeTypes
				err = rows.Scan(&coffeeType.CoffeeTypeId, &coffeeType.CoffeeType)
				allCoffeeTypes := append(ld.CoffeeTypes, coffeeType)
				ld.CoffeeTypes = allCoffeeTypes
				log.Println("added one coffeetype", allCoffeeTypes)
			}
		}
	}

	if ld.Accounttypeid != "" {
		z := strings.Split(ld.Accounttypeid, ",")
		for i, z := range z {
			log.Println("get accounts", i, z)
			sqlStatement := `SELECT accounttypeid, accounttype FROM dbo.cms_account_type_master where accounttypeid=$1`
			rows, err = db.Query(sqlStatement, z)

			if err != nil {
				log.Println(err)
				log.Println("unable to add account type", ld.ContactSalutationid)
			}

			defer rows.Close()
			for rows.Next() {
				var account AccountsInformation
				err = rows.Scan(&account.Accounttypeid, &account.Accounttype)
				allAccounts := append(ld.AccountTypes, account)
				ld.AccountTypes = allAccounts
				log.Println("added one account", allAccounts)
			}
		}
	}
	//---------------------Fetch Audit Log Info-------------------------------------//
	log.Println("Fetching Audit Log Info #")
	sqlStatementAI := `select u.username as createduser, ld.created_date,
					ld.description, v.username as modifieduser, ld.modified_date
	   				from dbo.auditlog_cms_leads_master_newpg ld
	   				inner join dbo.users_master_newpg u on ld.createdby=u.userid
	   				left join dbo.users_master_newpg v on ld.modifiedby=v.userid
	   				where ld.leadid=$1 order by logid desc limit 1`
	rowsAI, errAI := db.Query(sqlStatementAI, id.Id)
	log.Println("Audit Info Fetch Query Executed")
	if errAI != nil {
		log.Println("Audit Info Fetch Query failed")
		log.Println(errAI.Error())
		return events.APIGatewayProxyResponse{500, headers, nil, errAI.Error(), false}, nil
	}

	for rowsAI.Next() {
		var al AuditLogGCPO
		errAI = rowsAI.Scan(&al.CreatedUserid, &al.CreatedDate, &al.Description, &al.ModifiedUserid, &al.ModifiedDate)
		auditDetails := append(ld.AuditLogDetails, al)
		ld.AuditLogDetails = auditDetails
		log.Println("added one")

	}
	log.Println("Audit Details:", ld.AuditLogDetails)

	res, _ := json.Marshal(ld)
	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(getLeadDetailonHyperLink)
}
