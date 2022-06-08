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

type ApprQuotes struct {
	QuotationId       string `json:"quotation_id"`
	QuotationNo       string `json:"quotation_no"`
	QuotationDate     string `json:"quotation_date"`
	GCItemId          string `json:"item_id"`
	ConfirmedQuantity string `json:"qty"`
	Price             string `json:"price"`
}

type Input struct {
	Type   string `json:"type"`
	GcType string `json:"gc_type"`
	PoDate string `json:"po_date"`
}

func listApprovedQtLinesOnGc(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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
	var approvedqtlines []ApprQuotes
	var quotationDate, confQuantity, price sql.NullString
	var rows *sql.Rows
	if input.Type == "approvedqtlines" {
		sqlStatement := `SELECT q.quotationid, q.quotationno, q.quotationdate, 
		r.itemid, 
		SUM(l.coqkgs * r.perc * r.ratio) AS gcqty, 
		r.rmrate
FROM dbo.sales_quotation_master_new q
INNER JOIN dbo.sales_quotation_details_new as l ON q.quotationid = l.quotationid
INNER JOIN dbo.sales_quotation_details_rawmeterial_rates_new as r ON l.detid = r.detref
WHERE (r.is_selected_alt = 1) and r.itemid=$1 and q.quotationdate <=$2 and q.approvalstatus=1
GROUP BY q.quotationid,q.quotationno,q.quotationdate,r.itemid,r.rmrate`
		rows, err = db.Query(sqlStatement, input.GcType, input.PoDate)

		defer rows.Close()
		for rows.Next() {
			var ql ApprQuotes
			err = rows.Scan(&ql.QuotationId, &ql.QuotationNo, &quotationDate, &ql.GCItemId, &confQuantity, &price)
			ql.QuotationDate = quotationDate.String
			ql.ConfirmedQuantity = confQuantity.String
			ql.Price = price.String
			approvedqtlines = append(approvedqtlines, ql)
		}
	}

	res, _ := json.Marshal(approvedqtlines)
	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(listApprovedQtLinesOnGc)
}
