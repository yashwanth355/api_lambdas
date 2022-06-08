//Deployed with Pwd Changes
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	// "time"
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
	AccountID   int    `json:"accountid"`
	QuoteID     int    `json:"quoteid"`
	AccountName string `json:"accountname"`
	Status      string `json:"status"`
	CreatedBy   string `json:"createdby"`
	CreatedDate string `json:"createddate"`
}

func getAccountQuotes(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	var qd QuoteDetails
	err := json.Unmarshal([]byte(request.Body), &qd)
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

	var createdBy sql.NullString
	fmt.Println("Connected!")
	var rows *sql.Rows
	sqlStatementaq1 := `SELECT q.accountid, q.quoteid,a.accountname,s.status,u.username,q.createddate 
							FROM dbo.crm_quote_master q
							INNER JOIN dbo.accounts_master a ON q.accountid = a.accountid 
							INNER JOIN dbo.CMS_ALLSTATUS_MASTER s ON q.statusid = s.id
							LEFT join dbo.users_master_newpg u on u.userid=a.createduserid
						    where a.accountid=$1`

	rows, err = db.Query(sqlStatementaq1, qd.AccountID)
	var quoteDetailsList []QuoteDetails
	defer rows.Close()
	for rows.Next() {
		var quote QuoteDetails
		err = rows.Scan(&quote.AccountID, &quote.QuoteID, &quote.AccountName, &quote.Status, &createdBy, &quote.CreatedDate)
		quote.CreatedBy = createdBy.String
		quoteDetailsList = append(quoteDetailsList, quote)
	}
	res, _ := json.Marshal(quoteDetailsList)
	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(getAccountQuotes)
}
