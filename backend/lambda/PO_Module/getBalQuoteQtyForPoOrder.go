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

type QtyforPo struct {
	OrderQty string `json:"order_qty"`
}

type TotalQtyPoOrder struct {
	TotalQty sql.NullString `json:"total_qty"`
}

type Input struct {
	Type        string `json:"type"`
	QuotationId string `json:"quotation_no"`
}

func getBalQuoteQtyForPoOrder(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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

	var res []byte
	var rows *sql.Rows
	if input.Type == "getBalqtyforPo" {
		log.Println("get ordered qty on quotation id", input.Type)
		sqlStatement := `select sum(total_quantity) from dbo.pur_gc_po_con_master_newpg where quote_no=$1`
		rows, err = db.Query(sqlStatement, input.QuotationId)

		var g QtyforPo
		var t sql.NullString
		defer rows.Close()
		for rows.Next() {
			err = rows.Scan(&t)
		}
		g.OrderQty = t.String

		res, _ = json.Marshal(g)
	}

	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}

	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(getBalQuoteQtyForPoOrder)
}
