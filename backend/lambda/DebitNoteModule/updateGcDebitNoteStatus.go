package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

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

type Input struct {
	Type        string `json:"type"`
	Debitnoteid string `json:"debit_noteid"`
	UpdatedBy   string `json:"updated_by"`
}

func updateGcDebitNoteStatus(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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
	if input.Type == "changeToPendingStatus" {
		sqlStatement := `update dbo.inv_gc_debitnote_master_newpg set status ='2' where debitnoteid=$1`
		_, err := db.Query(sqlStatement, input.Debitnoteid)

		if err != nil {
			log.Println("Unable to change status to pending state")
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

	} else if input.Type == "changeToApprovedStatus" {

		sqlStatement := `update dbo.inv_gc_debitnote_master_newpg set status ='3' where debitnoteid=$1`
		_, err := db.Query(sqlStatement, input.Debitnoteid)

		if err != nil {
			log.Println("Unable to change status to in progess state")
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

	} else if input.Type == "changeToAccountVerifiedStatus" {

		sqlStatement := `update dbo.inv_gc_debitnote_master_newpg set status ='4' where debitnoteid=$1`
		_, err := db.Query(sqlStatement, input.Debitnoteid)

		if err != nil {
			log.Println("Unable to change status to closed state")
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

	}

	var createdBy, createdOn sql.NullString
	sqlStatementDT1 := `select createdby, created_date from dbo.auditlog_inv_gc_debitnote_master_newpg where debitnoteid=$1 order by logid desc limit 1`

	rows1, _ := db.Query(sqlStatementDT1, input.Debitnoteid)

	for rows1.Next() {
		err = rows1.Scan(&createdBy, &createdOn)
	}

	// Insert Audit Info
	log.Println("Entered Audit Module for DebitNote")
	sqlStatementADT := `INSERT INTO dbo.auditlog_inv_gc_debitnote_master_newpg(
		debitnoteid,createdby, created_date, description,modifiedby, modified_date)
					VALUES($1,$2,$3,$4,$5,$6)`

	description := "Debit Note status modified"
	_, errADT := db.Query(sqlStatementADT,
		input.Debitnoteid,
		createdBy,
		createdOn,
		description,
		input.UpdatedBy,
		time.Now().Format("2006-01-02"))

	log.Println("Audit Update Query Executed")
	if errADT != nil {
		log.Println("unable to update Audit Details", errADT)
	}

	return events.APIGatewayProxyResponse{200, headers, nil, string("Successfully changed"), false}, nil
}

func main() {
	lambda.Start(updateGcDebitNoteStatus)
}
