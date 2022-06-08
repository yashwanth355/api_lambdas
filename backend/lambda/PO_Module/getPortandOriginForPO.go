//Deployed-Aug22
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

type Origin struct {
	Origin string `json:"origin"`
}

type PortLoading struct {
	Port string `json:"Ports"`
}
type DestinationPort struct {
	DPortId string `json:"destination_port_id"`
	DPortName string `json:"destination_port_name"`
}

type Input struct {
	Type string `json:"type"`
}

func getPortandOriginForPO(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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
	var rows *sql.Rows
	if input.Type == "originDetails" {
		var items []Origin

		sqlStatement := `select distinct initcap(origin) from dbo.pur_gc_contract_master where origin != '';`
		rows, err = db.Query(sqlStatement)

		defer rows.Close()
		for rows.Next() {
			var item1 Origin
			err = rows.Scan(&item1.Origin)
			items = append(items, item1)
		}

		res, _ := json.Marshal(items)

		return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
	} else if input.Type == "portLoadingDetails" {
		var items []PortLoading

		sqlStatement := `select distinct initcap(poloading) from dbo.pur_gc_contract_master where poloading != '';`
		rows, err = db.Query(sqlStatement)

		defer rows.Close()
		for rows.Next() {
			var item1 PortLoading
			err = rows.Scan(&item1.Port)
			items = append(items, item1)
		}

		res, _ := json.Marshal(items)

		return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
	} else if input.Type == "destinationports" {
		log.Println("Finding destination ports")
		// var DPS = make([]DestinationPort, 0)
		
		sqlStatementDP := `select id,destinationportname from dbo.cms_portofdestination_master;`
		rows, err = db.Query(sqlStatementDP)
		var items []DestinationPort
		defer rows.Close()
		for rows.Next() {
			var item1 DestinationPort
			err = rows.Scan(&item1.DPortId,&item1.DPortName)
			items = append(items, item1)
		}

		res, _ := json.Marshal(items)

		return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
	}

	return events.APIGatewayProxyResponse{200, headers, nil, "", false}, nil
}

func main() {
	lambda.Start(getPortandOriginForPO)
}
