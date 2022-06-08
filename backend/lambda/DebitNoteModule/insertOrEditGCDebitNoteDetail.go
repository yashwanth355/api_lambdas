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

type DebitnoteDetails struct {
	Update          bool   `json:"update"`
	Debitnoteid     string `json:"debit_noteid"`
	Debitnoteidsno  int    `json:"debit_noteidsno"`
	Debitnoteno     string `json:"debit_noteno"`
	Debitnotedate   string `json:"debit_notedate"`
	VendorId        string `json:"vendor_id"`
	Remarks         string `json:"remarks"`
	InvoiceNo       string `json:"invoice_no"`
	InvoiceQuantity string `json:"invoice_qty"`
	EntityId        string `json:"entity_id"`
	Mrinid          string `json:"mrin_id"`
	MrinNo          string `json:"mrin_no"`
	Others          string `json:"others"`
	Itemid          string `json:"item_id"`
	DebitAmount     string `json:"debit_amount"`
	HscCode         string `json:"hsc_code"`
	CreatedOn       string `json:"created_on"`
	CreatedBy       string `json:"created_by"`
	UpdatedOn       string `json:"updated_on"`
	UpdatedBy       string `json:"updated_by"`
	//Special composition Info Section---------------------------
	Husk     string `json:"husk"`
	Quality  string `json:"quality"`
	Netrecd  string `json:"netrecd"`
	Moisture string `json:"moisture"`
	Stones   string `json:"stones"`
	/*Browns        int `json:"browns"`
	Blacks        int `json:"blacks"`
	BrokenBits    int `json:"broken_bits"`
	InsectedBeans int `json:"insected_beans"`
	Bleached      int `json:"bleached"`
	Sticks        int `json:"sticks"`
	BeansRetained int `json:"beans_retained"`*/
}

func insertOrEditGCDebitNoteDetail(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var debitnoteDetails DebitnoteDetails
	err := json.Unmarshal([]byte(request.Body), &debitnoteDetails)
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

	if debitnoteDetails.Update {
		sqlStatement1 := `UPDATE dbo.inv_gc_debitnote_master_newpg SET 
		vendorid=$1,
		remarks=$2,
		invoiceno=$3,
		invoice_qty=$4,
		entityid=$5, 
		mrinid=$6,
		mrinno=$7,
		itemid=$8,
		debitamount=$9,
		hsccode=$10, husk=$11, quality=$12,netrecd=$13,moisture=$14, debitnotedate=$15, others=$16, stones=$17 where debitnoteid=$18`

		_, err = db.Query(sqlStatement1,
			debitnoteDetails.VendorId, debitnoteDetails.Remarks, debitnoteDetails.InvoiceNo, debitnoteDetails.InvoiceQuantity,
			debitnoteDetails.EntityId, debitnoteDetails.Mrinid, debitnoteDetails.MrinNo,
			debitnoteDetails.Itemid, debitnoteDetails.DebitAmount, debitnoteDetails.HscCode, debitnoteDetails.Husk, debitnoteDetails.Quality,
			debitnoteDetails.Netrecd, debitnoteDetails.Moisture, debitnoteDetails.Debitnotedate, debitnoteDetails.Others, debitnoteDetails.Stones, debitnoteDetails.Debitnoteid)

		if err != nil {
			log.Println("unable to update debitnote details", err.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

		sqlStatementDT1 := `select createdby, created_date from dbo.auditlog_inv_gc_debitnote_master_newpg where debitnoteid=$1 order by logid desc limit 1`

		rows1, _ := db.Query(sqlStatementDT1, debitnoteDetails.Debitnoteid)

		for rows1.Next() {
			err = rows1.Scan(&debitnoteDetails.CreatedBy, &debitnoteDetails.CreatedOn)
		}

		// Insert Audit Info
		log.Println("Entered Audit Module for DebitNote")
		sqlStatementADT := `INSERT INTO dbo.auditlog_inv_gc_debitnote_master_newpg(
			debitnoteid,createdby, created_date, description,modifiedby, modified_date)
						VALUES($1,$2,$3,$4,$5,$6)`

		description := "Debit Note modified"
		_, errADT := db.Query(sqlStatementADT,
			debitnoteDetails.Debitnoteid,
			debitnoteDetails.CreatedBy,
			debitnoteDetails.CreatedOn,
			description,
			debitnoteDetails.UpdatedBy,
			time.Now().Format("2006-01-02"))

		log.Println("Audit Update Query Executed")
		if errADT != nil {
			log.Println("unable to update Audit Details", errADT)
		}

		return events.APIGatewayProxyResponse{200, headers, nil, string("Updated successfully"), false}, nil
	} else {

		sqlStatementDT1 := `select debitnoteidsno from dbo.inv_gc_debitnote_master_newpg order by debitnoteidsno desc limit 1`

		rows1, err := db.Query(sqlStatementDT1)

		for rows1.Next() {
			err = rows1.Scan(&debitnoteDetails.Debitnoteidsno)
		}

		debitnoteDetails.Debitnoteidsno = debitnoteDetails.Debitnoteidsno + 1
		debitnoteDetails.Debitnoteid = "FAC-" + strconv.Itoa(debitnoteDetails.Debitnoteidsno)
		debitnoteDetails.Debitnoteno = "CCL/GC/" + strconv.Itoa(debitnoteDetails.Debitnoteidsno) + "/" + strconv.Itoa(time.Now().Year()) + "-" + strconv.Itoa(time.Now().Year()+1)

		log.Println("Debitnoteidsno", debitnoteDetails.Debitnoteidsno, "Debitnoteid", debitnoteDetails.Debitnoteid, "Debitnoteno", debitnoteDetails.Debitnoteno)

		sqlStatement2 := `INSERT INTO dbo.inv_gc_debitnote_master_newpg (
			debitnotedate,
			vendorid,
			remarks, 
			invoiceno,
			invoice_qty,
			status,
			entityid,
			mrinid,
			mrinno,
			itemid,
			debitamount,
			hsccode,
			husk,quality, netrecd, moisture,
			debitnoteid,
			debitnoteidsno,
			debitnoteno, others, stones) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,$15,$16,$17,$18,$19,$20, $21)`
		log.Println("Debitnotedate", debitnoteDetails.Debitnotedate)
		rows2, err := db.Query(sqlStatement2,
			debitnoteDetails.Debitnotedate,
			debitnoteDetails.VendorId,
			debitnoteDetails.Remarks,
			debitnoteDetails.InvoiceNo,
			debitnoteDetails.InvoiceQuantity,
			"1",
			debitnoteDetails.EntityId,
			debitnoteDetails.Mrinid,
			debitnoteDetails.MrinNo,
			debitnoteDetails.Itemid,
			debitnoteDetails.DebitAmount, debitnoteDetails.HscCode, debitnoteDetails.Husk,
			debitnoteDetails.Quality, debitnoteDetails.Netrecd, debitnoteDetails.Moisture,
			debitnoteDetails.Debitnoteid,
			debitnoteDetails.Debitnoteidsno,
			debitnoteDetails.Debitnoteno, debitnoteDetails.Others, debitnoteDetails.Stones)

		defer rows2.Close()

		if err != nil {
			log.Println("Insert to debitnoteDetails request failed", err.Error())
		}

		// Insert Audit Info
		log.Println("Entered Audit Module for DebitNote")
		sqlStatementADT := `INSERT INTO dbo.auditlog_inv_gc_debitnote_master_newpg(
						debitnoteid,createdby, created_date, description)
						VALUES($1,$2,$3,$4)`

		description := "Debit Note Created"
		_, errADT := db.Query(sqlStatementADT,
			debitnoteDetails.Debitnoteid,
			debitnoteDetails.CreatedBy,
			time.Now().Format("2006-01-02"),
			description)

		log.Println("Audit Insert Query Executed")
		if errADT != nil {
			log.Println("unable to insert Audit Details", errADT)
		}

		return events.APIGatewayProxyResponse{200, headers, nil, string("Created successfully"), false}, nil
	}
}

func main() {
	lambda.Start(insertOrEditGCDebitNoteDetail)
}
