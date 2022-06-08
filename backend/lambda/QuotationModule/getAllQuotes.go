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

type QuoteDetails struct {
	QuoteNumber          int    `json:"quoteid"`
	QuoteGeneratedNumber string `json:"quotenumber"`
	AccountName          string `json:"accountname"`
	Status               string `json:"status"`
	Pendingwith          string `json:"pendingwith"`
	CreatedBy            string `json:"createdby"`
	CreatedDate          string `json:"createddate"`
}

type Input struct {
	Filter string `json:"filter"`
}

func getAllQuotes(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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
	var quoteDetailsList []QuoteDetails
	var pendingWithPerson, assignedToPerson sql.NullString
	var rows *sql.Rows
	var param string
	if input.Filter == "" {
		param = " order by createddate desc"
	} else {
		param = "where " + input.Filter + " order by createddate desc"
	}

	log.Println("filter Query :", param)

	sqlStatement1 := `select quoteid, quotenumber, accountname, status, pendingwith,createdby, createddate from dbo.QuotationGrid %s`
	rows, err = db.Query(fmt.Sprintf(sqlStatement1, param))
	defer rows.Close()
	for rows.Next() {
		var quote QuoteDetails
		err = rows.Scan(&quote.QuoteNumber, &quote.QuoteGeneratedNumber, &quote.AccountName, &quote.Status, &pendingWithPerson, &assignedToPerson, &quote.CreatedDate)
		quote.Pendingwith = pendingWithPerson.String
		quote.CreatedBy = assignedToPerson.String
		quoteDetailsList = append(quoteDetailsList, quote)
	}

	res, _ := json.Marshal(quoteDetailsList)
	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(getAllQuotes)
}
