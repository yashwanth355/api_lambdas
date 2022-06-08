//Deployed with pwd changes
//CHECKEDIN
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

type SampleRequestDetails struct {
	AccountID       int    `json:"accountid"`
	SampleRequestId string `json:"id"`
	AccountName     string `json:"accountname"`
	FirstName       string `json:"firstname"`
	LastName        string `json:"lastname"`
	// NoOfProducts    	int `json:"noofproducts"`
	SampleRequestDate string `json:"date"`
	CreatedBy         string `json:"createdby"`
	Status            string `json:"status"`
	ReqQC             string `json:"requesttoqcstatus"`
	RecQC             string `json:"receivedbyqcstatus"`
}

func getAccountSamples(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var srd SampleRequestDetails
	err := json.Unmarshal([]byte(request.Body), &srd)
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
	log.Println("get specific account Sample Requests")
	sqlStatementas1 := `SELECT 
							sr.accountid,
							sr.samplereqid,
							acc.accountname,
							cont.contactfirstname,
							cont.contactlastname,
							sr.createddate,
							sr.createduser,
							sr.masterstatus,
							sr.reqqc,sr.recqc
						from dbo.cms_sample_request_master sr
						inner join dbo.accounts_master acc on acc.accountid=sr.accountid
						inner join dbo.contacts_master cont on cont.contactid=sr.contactid
						where acc.accountid=$1`
	rows, err = db.Query(sqlStatementas1, srd.AccountID)

	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}

	var allSampReqs []SampleRequestDetails
	defer rows.Close()
	for rows.Next() {
		var sr SampleRequestDetails
		err = rows.Scan(&sr.AccountID, &sr.SampleRequestId, &sr.AccountName, &sr.FirstName, &sr.LastName,
			&sr.SampleRequestDate, &sr.CreatedBy, &sr.Status, &sr.ReqQC, &sr.RecQC)
		allSampReqs = append(allSampReqs, sr)
	}
	res, _ := json.Marshal(allSampReqs)
	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(getAccountSamples)
}
