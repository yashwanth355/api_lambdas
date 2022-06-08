//Deployed with pwd changes
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

type LeadDetails struct {
	LeadName string `json:"leadname"`
	Contact  string `json:"contact"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Website  string `json:"website"`
	UserName string `json:"username"`
}
type ERPContactDetails struct {
	CustomerName string `json:"customername"`
	ContactName  string `json:"contactname"`
	Country      string `json:"country"`
}

//Added comment
type LeadandERPContacts struct {
	LeadDetails       LeadDetails       `json:"leaddetails"`
	ERPContactDetails ERPContactDetails `json:"erpcontacts"`
}

func getContactsInLeadtoAcc(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var leadname LeadDetails
	err := json.Unmarshal([]byte(request.Body), &leadname)
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
	log.Println("Fetch the lead details in confirmation popup")
	var le LeadandERPContacts
	var userName, country sql.NullString
	sqlStatement1 := `select l.accountname,concat(l.contactfirstname,',',l.contactlastname) as contact,
					  l.phone,l.email,l.website,u.username 
					  from dbo.cms_leads_master as l
					  LEFT join dbo.users_master_newpg u on u.userid=l.createduserid
					  where l.accountname=$1`
	rows, err = db.Query(sqlStatement1, leadname.LeadName)
	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}
	// defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&le.LeadDetails.LeadName, &le.LeadDetails.Contact, &le.LeadDetails.Phone,
			&le.LeadDetails.Email, &le.LeadDetails.Website, &userName)
		le.LeadDetails.UserName = userName.String
	}
	var parameter string
	parameter = "'%" + leadname.LeadName + "%'"
	log.Println("Fetch the existing ERP Contacts in confirmation Popup")
	sqlStatementERPCon := `select con.custname,
							concat(con.contactfirstname,' ',con.contactlastname) as contactname,
							abm.country 
							from dbo.contacts_master con
							left join dbo.accounts_billing_address_master abm
							on abm.accountid=con.accountid 
							where con.custname ilike %s`
	rows, err = db.Query(fmt.Sprintf(sqlStatementERPCon, parameter))
	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&le.ERPContactDetails.CustomerName, &le.ERPContactDetails.ContactName, &country)
		le.ERPContactDetails.Country = country.String
	}
	res, _ := json.Marshal(le)
	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(getContactsInLeadtoAcc)
}
