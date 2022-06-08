package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

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

type DebitNoteDetails struct {
	Debitnoteid     string `json:"debitnoteid"`
	Debitnoteno     string `json:"debitnoteno"`
	Debitnotedate   string `json:"debitnotedate"`
	InvoiceNo       string `json:"invoiceno"`
	InvoiceQuantity string `json:"invoice_qty"`
	Status          string `json:"status"`
	DebitAmount     string `json:"debitamount"`
	MrinNO          string `json:"mrinno"`
	VendorName      string `json:"vendorname"`
	Entity          string `json:"entity"`
	Husk            string `json:"husk"`
	Quality         string `json:"quality"`
	Netrecd         string `json:"netrecd"`
	Moisture        string `json:"moisture"`
}

type Input struct {
	Filter string `json:"filter"`
}

func listGCDebitNoteDetails(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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
		param = " order by debitnoteidsno desc"
	} else if input.Filter == "status = 'Released'" || strings.Index(input.Filter, "Released") != -1 {
		param = "where is_released = true order by debitnoteidsno desc"
	} else {
		param = "where " + input.Filter + " order by debitnoteidsno desc"
	}
	log.Println("filter Query :", param)
	var rows *sql.Rows
	var allDebitNoteDetails []DebitNoteDetails
	sqlStatement1 := `select debitnoteid, debitnotedate, debitnoteno, invoiceno, invoice_qty, status, debitamount,husk,
	quality,netrecd, moisture, vendorname, entityname, mrinno from dbo.DebitnoteGrid %s`
	rows, err = db.Query(fmt.Sprintf(sqlStatement1, param))
	var invoiceNo, invoiceQuantity, debitAmount, husk, quality, netRecd, moisture, vendorName, entity, mrinNo sql.NullString
	defer rows.Close()
	for rows.Next() {
		var debit DebitNoteDetails
		err = rows.Scan(&debit.Debitnoteid, &debit.Debitnotedate, &debit.Debitnoteno, &invoiceNo, &invoiceQuantity,
			&debit.Status, &debitAmount, &husk,
			&quality, &netRecd, &moisture, &vendorName, &entity, &mrinNo)
		debit.InvoiceNo = invoiceNo.String
		debit.InvoiceQuantity = invoiceQuantity.String
		debit.DebitAmount = debitAmount.String
		debit.Husk = husk.String
		debit.Quality = quality.String
		debit.Netrecd = netRecd.String
		debit.Moisture = moisture.String
		debit.VendorName = vendorName.String
		debit.Entity = entity.String
		debit.MrinNO = mrinNo.String

		allDebitNoteDetails = append(allDebitNoteDetails, debit)
	}

	res, _ := json.Marshal(allDebitNoteDetails)
	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(listGCDebitNoteDetails)
}
