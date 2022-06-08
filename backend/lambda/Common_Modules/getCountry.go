//Deployed with pwd changes
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

type country struct {
	CountryId string `json:"countryid"`
	Country   string `json:"country"`
}

func getCountry(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
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

	rows, err := db.Query(`SELECT countryid, countryname FROM dbo.countries_master`)

	var countryList []country
	defer rows.Close()
	for rows.Next() {
		var cnty country
		err = rows.Scan(&cnty.CountryId, &cnty.Country)
		countryList = append(countryList, cnty)
	}
	res, _ := json.Marshal(countryList)
	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(getCountry)
}
