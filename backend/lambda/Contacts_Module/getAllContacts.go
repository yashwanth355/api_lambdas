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

type ContactDetails struct {
	AccountName      string `json:"accountname"`
	AccountID        int    `json:"accountid"`
	ContactFirstName string `json:"contactfirstname"`
	ContactLastName  string `json:"contactlastname"`
	ContactEmail     string `json:"contactemail"`
	ContactPhone     string `json:"contactphone"`
	ContactID        int    `json:"contactid"`
}
type Input struct {
	Filter string `json:"filter"`
}

func getAllContacts(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlInfo)
	var input Input
	err1 := json.Unmarshal([]byte(request.Body), &input)
	if err1 != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err1.Error(), false}, nil
	}

	defer db.Close()

	// check db
	err = db.Ping()

	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}

	fmt.Println("Connected!")
	var param string
	if input.Filter == "" {
		param = " order by createddate desc"
	} else {
		param = "where " + input.Filter + " order by createddate desc"
	}
	log.Println("filter Query :", param)

	var rows *sql.Rows
	var email, contactLastName, contactPhone sql.NullString
	sqlStatement1 := `SELECT accountid,accountname,contactid,contactfirstname,contactlastname,
						contactemail,contactphone FROM dbo.ContactGrid %s`
	rows, err = db.Query(fmt.Sprintf(sqlStatement1, param))
	print(sqlStatement1)
	var contactDetailsList []ContactDetails
	defer rows.Close()
	for rows.Next() {
		var contact ContactDetails
		err = rows.Scan(&contact.AccountID, &contact.AccountName, &contact.ContactID, &contact.ContactFirstName, &contactLastName, &email, &contactPhone)
		contact.ContactLastName = contactLastName.String
		contact.ContactEmail = email.String
		contact.ContactPhone = contactPhone.String
		contactDetailsList = append(contactDetailsList, contact)
	}
	log.Println(contactDetailsList)
	res, _ := json.Marshal(contactDetailsList)
	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(getAllContacts)
}
