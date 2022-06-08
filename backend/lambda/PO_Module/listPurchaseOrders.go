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

type PurchaseOrderDetails struct {
	PoId           string `json:"poid"`
	PoNo           string `json:"pono"`
	PoDate         string `json:"podate"`
	ApprovalStatus string `json:"status"`
	Vendor         string `json:"vendorname"`
	VendorTypeId   string `json:"vendortype"`
	Category       string `json:"pocat"`
	Currency       string `json:"currencycode"`
	PoValue        string `json:"povalue"`
	TotalQuantity  string `json:"total_quantity"`
	GCItemName     string `json:"gc_itemname"`
}

type Input struct {
	Filter string `json:"filter"`
}

func listPurchaseOrders(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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
		param = "dbo.PurchaseOrdersGrid order by poidsno desc"
	} else if strings.Contains(input.Filter, "advance"){
		param = "dbo.gcpo_advancedpaymentsgrid where " + input.Filter + " order by poidsno desc"
		
	} else {
		param = "dbo.PurchaseOrdersGrid where " + input.Filter + " order by poidsno desc"
	}

	log.Println("filter Query :", param)
	var rows *sql.Rows
	var allPurchaseOrderDetails []PurchaseOrderDetails
	sqlStatement1 := `select poid,pono,podate,vendorname,status,pocat,vendortype,currencycode,povalue,total_quantity,gc_itemname from %s`
	rows, err = db.Query(fmt.Sprintf(sqlStatement1, param))
	log.Println(fmt.Sprintf(sqlStatement1,param))
	var poValue, quantity, currency, itemName sql.NullString
	defer rows.Close()
	for rows.Next() {
		var po PurchaseOrderDetails
		err = rows.Scan(&po.PoId, &po.PoNo, &po.PoDate, &po.Vendor,
			&po.ApprovalStatus, &po.Category, &po.VendorTypeId, &currency, &poValue, &quantity, &itemName)
		po.PoValue = poValue.String
		po.Currency = currency.String
		po.TotalQuantity = quantity.String
		po.GCItemName = itemName.String
		allPurchaseOrderDetails = append(allPurchaseOrderDetails, po)
	}

	res, _ := json.Marshal(allPurchaseOrderDetails)
	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(listPurchaseOrders)
}
