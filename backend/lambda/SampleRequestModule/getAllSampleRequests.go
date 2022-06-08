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
	SampleReqId       string `json:"samplereqid"`
	SampleRequestNo   string `json:"samplereqno"`
	AccountName       string `json:"accountname"`
	FirstName         string `json:"contactfirstname"`
	LastName          string `json:"contactlastname"`
	Products          int    `json:"products"`
	SampleRequestDate string `json:"createddate"`
	CreatedBy         string `json:"createdbyusername"`
	Status            string `json:"status"`
	ReqQC             bool   `json:"requesttoqcstatus"`
	RecQC             bool   `json:"receivedqcstatus"`
}

type Input struct {
	Filter string `json:"filter"`
}

func getAllSampleRequests(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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
		param = " order by cast(samplereqid as int) desc"
	} else {
		param = "where " + input.Filter + " order by cast(samplereqid as int) desc"
	}

	log.Println("filter Query :", param)
	var allSampReqs []SampleRequestDetails

	var name sql.NullString

	var rows *sql.Rows
	log.Println("get Sample Requests as per the filter")

	sqlStatement1 := `select samplereqid, samplereqno, accountname, contactfirstname,
	 contactlastname, products, createddate, createdbyusername, status, requesttoqcstatus, receivedqcstatus from dbo.SampleRequestGrid %s`
	rows, err = db.Query(fmt.Sprintf(sqlStatement1, param))
	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}

	defer rows.Close()
	for rows.Next() {
		var sr SampleRequestDetails
		err = rows.Scan(&sr.SampleReqId, &sr.SampleRequestNo, &sr.AccountName, &sr.FirstName, &sr.LastName, &sr.Products,
			&sr.SampleRequestDate, &name, &sr.Status, &sr.ReqQC, &sr.RecQC)
		sr.CreatedBy = name.String
		allSampReqs = append(allSampReqs, sr)
	}

	res, _ := json.Marshal(allSampReqs)
	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(getAllSampleRequests)
}
