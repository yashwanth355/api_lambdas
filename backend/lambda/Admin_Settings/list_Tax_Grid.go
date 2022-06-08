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

type TaxMasterDetails struct {
	TaxName         string             `json:"tax_name"`
	TaxPercentage 	string             `json:"tax_percentage"`
	IsActive		bool 			   `json:"isactive"`
	TaxId        string             `json:"tax_id"`
}

type Input struct {
	Filter string `json:"filter"`
}

func list_Tax_Grid(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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
		param = " order by taxidsno desc"
	} else {
		param = "where " + input.Filter + " order by taxidsno desc"
	}

	log.Println("filter Query :", param)
	var rows *sql.Rows
	var list []TaxMasterDetails
	
	sqlStatement1 := `select taxid,tax_name,tax_percentage,status from dbo.TaxMasterGrid %s`
	rows, err = db.Query(fmt.Sprintf(sqlStatement1, param))
	defer rows.Close()
	for rows.Next() {
		var list1 TaxMasterDetails
		err = rows.Scan(&list1.TaxId, &list1.TaxName, &list1.TaxPercentage,&list1.IsActive)
		list = append(list, list1)
	}
	res, _ := json.Marshal(list)
	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(list_Tax_Grid)
}
