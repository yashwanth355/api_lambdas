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

type MrinRequestDetails struct {
	Mrinid          string `json:"mrinid"`
	Mrinno          string `json:"mrinno"`
	Mrindate        string `json:"mrindate"`
	VendorName      string `json:"vendorname"`
	PoNumber        string `json:"pono"`
	PoDate          string `json:"podate"`
	InvoiceNo       string `json:"invoiceno"`
	InvoiceDate     string `json:"invoicedate"`
	RequestApproval string `json:"reqapproval"`
	EntityName      string `json:"entityname"`
	Status          string `json:"status"`
	MrinValue       string `json:"mrinvalue"`
	VendorTypeId   string `json:"vendortype"`
}
type Input struct {
	Filter string `json:"filter"`
}

func listMrins(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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
		param = " order by mrinidsno desc"
	} else {
		param = "where " + input.Filter + " order by mrinidsno desc"
	}

	log.Println("filter Query :", param)
	var rows *sql.Rows
	var allMrinRequestDetails []MrinRequestDetails
	var poDate, invoiceNo, mrinValue,vendor_type sql.NullString
	sqlStatement1 := `select mrinid, mrinno, mrindate,entityname, pono,
	podate, vendorname, invoiceno, invoicedate,reqapproval, status, mrinvalue,vendortype from dbo.MrinGrid %s`
	rows, err = db.Query(fmt.Sprintf(sqlStatement1, param))
	defer rows.Close()
	for rows.Next() {
		var po MrinRequestDetails
		err = rows.Scan(&po.Mrinid, &po.Mrinno, &po.Mrindate, &po.EntityName,
			&po.PoNumber, &poDate, &po.VendorName, &invoiceNo, &po.InvoiceDate, &po.RequestApproval, &po.Status, &mrinValue,&vendor_type)
		po.PoDate = poDate.String
		po.InvoiceNo = invoiceNo.String
		po.MrinValue = mrinValue.String
		po.VendorTypeId = vendor_type.String
		allMrinRequestDetails = append(allMrinRequestDetails, po)
	}
	res, _ := json.Marshal(allMrinRequestDetails)
	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}
func main() {
	lambda.Start(listMrins)
}
