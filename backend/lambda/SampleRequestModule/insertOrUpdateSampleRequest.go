package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
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
func NewNullString(s string) sql.NullString {
	if len(s) == 0 {
		return sql.NullString{}
	}
	return sql.NullString{
		String: s,
		Valid:  true,
	}
}
type SampleDetails struct {
	Update          bool   `json:"update"`
	SampleId        int    `json:"sample_id"`
	SampleReqNo     string `json:"sample_reqno"`
	SampleReqDate   string `json:"sample_reqdate"`
	LastSampleId    int    `json:"lastsample_id"`
	Accountid       string `json:"account_id"`
	ContactId       string `json:"contact_id"`
	ShippingAddress string `json:"shipping_address"`
	Remarks         string `json:"remarks"`
	CreatedDate     string `json:"createddate"`
	UserName        string `json:"username"`
	CreatedUserid   string `json:"createduserid"`
	ModifiedDate    string `json:"modifieddate"`
	ModifiedUserid  string `json:"modifieduserid"`
	Masterstatus    string `json:"masterstatus"`
}

func insertOrUpdateSampleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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

		var reqqc, recivedbyqc bool
		if sd.Masterstatus == "Pending with QC" {
			reqqc = true
		} else if sd.Masterstatus == "Approved" {
			recivedbyqc = true
			reqqc = true
		}
		sqlStatement1 := `UPDATE dbo.cms_sample_request_master SET 
		accountid=$1,
		contactid=$2,
		shipping_address=$3,
		remarks=$4,
		lastmodified=$5,
		lastmodifiedby=$6,
		lastmodifieduser=$7, masterstatus=$8, reqqc=$9, recqc=$10 where samplereqid=$11`

		rows, err = db.Query(sqlStatement1, sd.Accountid, NewNullString(sd.ContactId), sd.ShippingAddress,
			sd.Remarks, sd.ModifiedDate, sd.ModifiedUserid, sd.UserName, sd.Masterstatus, reqqc, recivedbyqc, sd.SampleId)

		if err != nil {
			log.Println("unable to update sample request", err.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

		sqlStatementDT1 := `select createdby, created_date from dbo.auditlog_cms_sample_request_master_newpg where samplereqid=$1 order by logid desc limit 1`

		rows1, _ := db.Query(sqlStatementDT1, sd.SampleId)

		for rows1.Next() {
			err = rows1.Scan(&sd.CreatedUserid, &sd.CreatedDate)
		}

		// Insert Audit Info
		log.Println("Entered Audit Module for Sample request")
		sqlStatementADT := `INSERT INTO dbo.auditlog_cms_sample_request_master_newpg(
			samplereqid,createdby, created_date, description,modifiedby, modified_date, status)
						VALUES($1,$2,$3,$4,$5,$6,$7)`

		description := "Sample request modified"
		_, errADT := db.Query(sqlStatementADT,
			sd.SampleId,
			sd.CreatedUserid,
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

		sqlStatementDT1 := `select samplereqid from dbo.cms_sample_request_master order by cast(samplereqid as int) desc limit 1`

		rows1, err := db.Query(sqlStatementDT1)

		// var po InputAdditionalDetails
		for rows1.Next() {
			err = rows1.Scan(&sd.LastSampleId)
		}

		sd.SampleId = sd.LastSampleId + 1
		sd.SampleReqNo = time.Now().Format("20060102") + "-" + strconv.Itoa(sd.SampleId)
		sd.SampleReqDate = time.Now().Format("2006-01-02")

		log.Println("sampleid", sd.SampleId, "samplereqno", sd.SampleReqNo, "samplereqdate", sd.SampleReqDate)

		sqlStatement2 := `INSERT INTO dbo.cms_sample_request_master (
			samplereqid,
			samplereqdate,
			samplereqno,
			accountid,
			contactid,
			shipping_address,
			createddate,
	        createduser,
			createdby,
			remarks,
			masterstatus,
			recordtypeid, reqqc, recqc, isactive) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`

		rows, err = db.Query(sqlStatement2,
			sd.SampleId,
			sd.SampleReqDate,
			sd.SampleReqNo,
			sd.Accountid,
			NewNullString(sd.ContactId),
			sd.ShippingAddress,
			sd.CreatedDate,
			sd.UserName,
			sd.CreatedUserid,
			sd.Remarks,
			sd.Masterstatus, "2", false, false, true)

		if err != nil {
			log.Println("Insert to sample request failed", err.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

		// Insert Audit Info
		log.Println("Entered Audit Module for sample request")
		sqlStatementADT := `INSERT INTO dbo.auditlog_cms_sample_request_master_newpg(
								samplereqid,createdby, created_date, description, status)
								VALUES($1,$2,$3,$4, $5)`

		description := "Sample request created"
		_, errADT := db.Query(sqlStatementADT,
			sd.SampleId,
			sd.CreatedUserid,
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
	lambda.Start(insertOrUpdateSampleRequest)
}
