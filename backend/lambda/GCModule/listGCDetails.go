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

type GCDetails struct {
	ItemId       string `json:"itemid"`
	ItemCode     string `json:"itemcode"`
	SCode        string `json:"s_code"`
	ItemName     string `json:"itemname"`
	GroupId      string `json:"groupid"`
	Groupname    string `json:"groupname"`
	Uom          string `json:"uom"`
	CategoryName string `json:"categoryname"`
	DisplayInPo  bool   `json:"display_inpo"`
	CoffeeType   string `json:"coffee_type"`
}

type Input struct {
	Filter string `json:"filter"`
}

func listGCDetails(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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
		param = " order by itemidsno desc"
	} else {
		param = "where " + input.Filter + " order by itemidsno desc"
	}

	log.Println("filter Query :", param)
	var rows *sql.Rows
	var allGCDetails []GCDetails
	sqlStatement1 := `select itemid, itemcode, s_code, itemname, categoryname, uom, groupid, groupname, display_inpo, coffee_type from dbo.GCGrid %s`
	rows, err = db.Query(fmt.Sprintf(sqlStatement1, param))
	fmt.Println(sqlStatement1)
	defer rows.Close()
	for rows.Next() {
		var gc GCDetails
		err = rows.Scan(&gc.ItemId, &gc.ItemCode, &gc.SCode, &gc.ItemName, &gc.CategoryName, &gc.Uom, &gc.GroupId, &gc.Groupname,
			&gc.DisplayInPo, &gc.CoffeeType)
		allGCDetails = append(allGCDetails, gc)
	}
	res, _ := json.Marshal(allGCDetails)
	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(listGCDetails)
}
