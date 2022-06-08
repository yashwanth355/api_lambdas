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

type Leads struct {
	Leadid            string `json:"leadid"`
	LeadName          string `json:"accountname"`
	Aliases           string `json:"aliases"`
	Firstname         string `json:"contactfirstname"`
	Lastname          string `json:"contactlastname"`
	ContactMobile     string `json:"contact_mobile"`
	Email             string `json:"email"`
	ProfileCompletion int    `json:"leadscore"`
	Status            string `json:"masterstatus"`
}

type Input struct {
	Filter string `json:"filter"`
}

func getLeadsInfo(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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
	var param string
	if input.Filter == "" {
		param = " ORDER BY createddate desc"
	} else {
		param = "where " + input.Filter + " ORDER BY createddate desc"
	}

	log.Println("filter Query :", param)
	var rows *sql.Rows
	sqlStatement1 := fmt.Sprintf("SELECT leadid,accountname,aliases,contactfirstname,contactlastname,contact_mobile,email,leadscore,masterstatus FROM dbo.LeadsGrid %s", param)
	rows, err = db.Query(sqlStatement1)
	var aliases, firstName, lastName, mobile, email, status sql.NullString
	print(sqlStatement1)
	var allLeads []Leads
	defer rows.Close()
	for rows.Next() {
		var lead Leads
		err = rows.Scan(&lead.Leadid, &lead.LeadName, &aliases, &firstName, &lastName, &mobile, &email, &lead.ProfileCompletion, &status)
		lead.Aliases = aliases.String
		lead.Firstname = firstName.String
		lead.Lastname = lastName.String
		lead.ContactMobile = mobile.String
		lead.Email = email.String
		lead.Status = status.String
		allLeads = append(allLeads, lead)
	}
	res, _ := json.Marshal(allLeads)
	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(getLeadsInfo)
}
