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

type SampleDetails struct {
	Update             bool   `json:"update"`
	LineItemId         int    `json:"lineitem_id"`
	SampleReqId        string `json:"sample_reqid"`
	SerialCount        int    `json:"serial_no"`
	SampleCatId        string `json:"sample_catid"`
	LastLineItemId     int    `json:"lastitem_id"`
	ProductId          string `json:"product_id"`
	Description        string `json:"description"`
	TargetPriceEnabled bool   `json:"targetprice_enabled"`
	TargetPrice        string `json:"target_price"`
	CreatedDate        string `json:"created_date"`
	CreatedByUser      string `json:"created_byuser"`
	CreatedbByUserid   string `json:"created_byuserid"`
	ModifiedDate       string `json:"modified_date"`
	ModifiedByUser     string `json:"modified_byuser"`
	ModifiedUserid     string `json:"modified_byuserid"`
	Masterstatus       string `json:"masterstatus"`
}

func insertOrUpdateSampleLineItem(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var sd SampleDetails
	err := json.Unmarshal([]byte(request.Body), &sd)
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

	if sd.Update {

		sqlStatement1 := `UPDATE dbo.cms_sample_request_details SET 
		samplereqid=$1,
		productid=$2,
		description=$3,
		targetprice=$4,
		lastmodifieddate=$5,
		lastmodifiedby=$6,
		lastmodifieduser=$7, samplecatid=$8 where lineitemid=$9`

		if !sd.TargetPriceEnabled || sd.TargetPrice == "" {
			sd.TargetPrice = "No target price"
		}
		rows, err = db.Query(sqlStatement1, sd.SampleReqId, sd.ProductId, sd.Description,
			sd.TargetPrice, sd.ModifiedDate, sd.ModifiedUserid, sd.ModifiedByUser, sd.SampleCatId, sd.LineItemId)

		if err != nil {
			log.Println("unable to update sample line item", err.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

		sqlStatementDT1 := `select createdby, created_date from dbo.auditlog_cms_sample_request_details_newpg where lineitemid=$1 order by logid desc limit 1`

		rows1, _ := db.Query(sqlStatementDT1, sd.LineItemId)

		for rows1.Next() {
			err = rows1.Scan(&sd.CreatedbByUserid, &sd.CreatedDate)
		}

		// Insert Audit Info
		log.Println("Entered Audit Module for Sample line item")
		sqlStatementADT := `INSERT INTO dbo.auditlog_cms_sample_request_details_newpg(
			lineitemid,createdby, created_date, description,modifiedby, modified_date, status)
						VALUES($1,$2,$3,$4,$5,$6,$7)`

		description := "Sample line item modified"
		_, errADT := db.Query(sqlStatementADT,
			sd.LineItemId,
			sd.CreatedbByUserid,
			sd.CreatedDate,
			description,
			sd.ModifiedUserid,
			time.Now().Format("2006-01-02"), sd.Masterstatus)

		log.Println("Audit Update Query Executed")
		if errADT != nil {
			log.Println("unable to update Audit Details", errADT)
		}

		res, _ := json.Marshal(rows)
		return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
	} else {

		sqlStatementDT1 := `select lineitemid from dbo.cms_sample_request_details order by lineitemid desc limit 1`

		rows1, err := db.Query(sqlStatementDT1)

		for rows1.Next() {
			err = rows1.Scan(&sd.LastLineItemId)
		}

		sqlStatementDT2 := `select count(samplereqid) from dbo.cms_sample_request_details where samplereqid=$1`

		rows2, err := db.Query(sqlStatementDT2, sd.SampleReqId)

		var srNo int

		for rows2.Next() {
			err = rows2.Scan(&sd.SerialCount)
		}

		sd.LineItemId = sd.LastLineItemId + 1
		srNo = sd.SerialCount + 1

		log.Println("lineitemid", sd.LineItemId, "serial no:", srNo, "serial count:", sd.SerialCount)

		if !sd.TargetPriceEnabled || sd.TargetPrice == "" {
			sd.TargetPrice = "No target price"
		}

		sqlStatement2 := `INSERT INTO dbo.cms_sample_request_details (
			lineitemid,
			samplereqid,
			productid,
			description,
			targetprice,
			samplecatid,
			createddate,
	        createduser,
			createdby,sno, isactive) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

		rows, err = db.Query(sqlStatement2,
			sd.LineItemId,
			sd.SampleReqId,
			sd.ProductId,
			sd.Description,
			sd.TargetPrice,
			sd.SampleCatId,
			sd.CreatedDate,
			sd.CreatedByUser,
			sd.CreatedbByUserid, srNo, true)

		if err != nil {
			log.Println("Insert to sample line item failed", err.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

		// Insert Audit Info
		log.Println("Entered Audit Module for sample line item")
		sqlStatementADT := `INSERT INTO dbo.auditlog_cms_sample_request_details_newpg(
								lineitemid,createdby, created_date, description, status)
								VALUES($1,$2,$3,$4, $5)`

		description := "Sample line item created"
		_, errADT := db.Query(sqlStatementADT,
			sd.LineItemId,
			sd.CreatedbByUserid,
			time.Now().Format("2006-01-02"),
			description, sd.Masterstatus)

		log.Println("Audit Insert Query Executed")
		if errADT != nil {
			log.Println("unable to insert Audit Details", errADT)
		}

		res, _ := json.Marshal(rows)
		return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
	}
}

func main() {
	lambda.Start(insertOrUpdateSampleLineItem)
}
