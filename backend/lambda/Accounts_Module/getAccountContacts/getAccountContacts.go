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

type ContactDetails struct {
	AccountID int        `json:"accountid"`
	FirstName NullString `json:"firstname"`
	LastName  NullString `json:"lastname"`
	Email     NullString `json:"email"`
	ContactID NullString `json:"contactid"`
}

type NullString struct {
	sql.NullString
}

// MarshalJSON for NullString
func (ns *NullString) MarshalJSON() ([]byte, error) {
	if !ns.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(ns.String)
}

func getAccountContacts(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var cd ContactDetails
	err := json.Unmarshal([]byte(request.Body), &cd)
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
	log.Println(cd.AccountID)
	sqlStatementc1 := `SELECT c.contactid,c.contactfirstname,c.contactlastname,c.contactemail 
							FROM dbo.contacts_master c 
							INNER JOIN dbo.accounts_master a ON c.accountid = a.accountid
							where a.accountid=$1`
	rows, err = db.Query(sqlStatementc1, cd.AccountID)

	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}

	var accCon []ContactDetails
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&cd.ContactID, &cd.FirstName, &cd.LastName, &cd.Email)
		accCon = append(accCon, cd)
	}
	res, _ := json.Marshal(accCon)
	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(getAccountContacts)
}
