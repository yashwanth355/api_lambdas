//CHECKEDIN
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
	Type      string `json:"type"`
	QuoteId   string `json:"quote_id"`
	UpdatedBy string `json:"updated_by"`
}

func updateQuotationStatus(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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
	var auditStatus string

	fmt.Println("Connected!")
	log.Println("update Quoteline item")
	if input.Type == "requestforprice" {
		sqlStatement1 := `UPDATE dbo.crm_quote_master SET 
	statusid=$1,
	modifiedby=$2,
	modifieddate=$3,
	ispricerequested=$4 where quoteid=$5`

		_, err = db.Query(sqlStatement1,
			"2",
			input.UpdatedBy,
			time.Now(),
			1, input.QuoteId)
		auditStatus = "Pending with GMC"
		if err != nil {
			log.Println(err)
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

	} else if input.Type == "quotesubmit" {
		sqlStatement1 := `UPDATE dbo.crm_quote_master SET 
	statusid=$1,
	modifiedby=$2,
	modifieddate=$3
	where quoteid=$4`

		_, err = db.Query(sqlStatement1,
			"4",
			input.UpdatedBy,
			time.Now(),
			input.QuoteId)
		auditStatus = "Quote submitted"
		if err != nil {
			log.Println(err)
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

	} else if input.Type == "quoteapprove" {
		sqlStatement1 := `UPDATE dbo.crm_quote_master SET 
		statusid=$1,
		modifiedby=$2,
		modifieddate=$3
		where quoteid=$4`

		_, err = db.Query(sqlStatement1,
			"5",
			input.UpdatedBy,
			time.Now(),
			input.QuoteId)
		auditStatus = "Quote approved"
		if err != nil {
			log.Println(err)
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

	} else if input.Type == "quotereject" {
		sqlStatement1 := `UPDATE dbo.crm_quote_master SET 
		statusid=$1,
		modifiedby=$2,
		modifieddate=$3
		where quoteid=$4`

		_, err = db.Query(sqlStatement1,
			"6",
			input.UpdatedBy,
			time.Now(),
			input.QuoteId)
		auditStatus = "Quote rejected"
		if err != nil {
			log.Println(err)
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

	} else if input.Type == "bidsubmitted" {
		sqlStatement1 := `UPDATE dbo.crm_quote_master SET 
	statusid=$1,
	modifiedby=$2,
	modifieddate=$3
	where quoteid=$4`

		_, err = db.Query(sqlStatement1,
			"12",
			input.UpdatedBy,
			time.Now(),
			input.QuoteId)
		auditStatus = "Bid submitted to GMC"
		if err != nil {
			log.Println(err)
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

	} else if input.Type == "quoteapprovedbygmc" {
		sqlStatement1 := `UPDATE dbo.crm_quote_master SET 
	statusid=$1,
	modifiedby=$2,
	modifieddate=$3
	where quoteid=$4`

		_, err = db.Query(sqlStatement1,
			"11",
			input.UpdatedBy,
			time.Now(),
			input.QuoteId)
		auditStatus = "Quote approved by GMC"
		if err != nil {
			log.Println(err)
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

	} else if input.Type == "quoterejectedbygmc" {
		sqlStatement1 := `UPDATE dbo.crm_quote_master SET 
	statusid=$1,
	modifiedby=$2,
	modifieddate=$3
	where quoteid=$4`

		_, err = db.Query(sqlStatement1,
			"10",
			input.UpdatedBy,
			time.Now(),
			input.QuoteId)
		auditStatus = "Quote rejected by GMC"
		if err != nil {
			log.Println(err)
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}
	}

	sqlStatementDT1 := `select createdby, created_date from dbo.auditlog_crm_quote_master_newpg where quoteid=$1 order by logid desc limit 1`

	rows1, _ := db.Query(sqlStatementDT1, input.QuoteId)
	var createdUser, createdDate sql.NullString

	for rows1.Next() {
		err = rows1.Scan(&createdUser, &createdDate)
	}

	// Insert Audit Info
	log.Println("Entered Audit Module for Quotation")
	sqlStatementADT := `INSERT INTO dbo.auditlog_crm_quote_master_newpg(
			quoteid,createdby, created_date, description,modifiedby, modified_date, status)
						VALUES($1,$2,$3,$4,$5,$6,$7)`

	description := "Quotation status modified"
	_, errADT := db.Query(sqlStatementADT,
		input.QuoteId,
		createdUser.String,
		createdDate.String,
		description,
		input.UpdatedBy,
		time.Now().Format("2006-01-02"), auditStatus)

	log.Println("Audit Update Query Executed")
	if errADT != nil {
		log.Println("unable to update Audit Details", errADT)
	}
	return events.APIGatewayProxyResponse{203, headers, nil, string("success"), false}, nil
}

func main() {
	lambda.Start(updateQuotationStatus)
}
