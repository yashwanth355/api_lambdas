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

type listsupplier struct {
	Name          string `json:"vendorname"`
	ContactPerson string `json:"contactname"`
	Phone         string `json:"phone"`
	City          string `json:"city"`
	State         string `json:"state"`
	Country       string `json:"country"`
	Group         string `json:"groupname"`
	Suppliers     string `json:"suppliers"`
	VendorId      string `json:"vendorid"`
	IsActive 		bool `json:"isactive"`
}

type Input struct {
	Filter string `json:"filter"`
}

func listSuppliers(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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
		param = " order by vendoridsno desc"
	} else {
		param = "where " + input.Filter + " order by vendoridsno desc"
	}

	log.Println("filter Query :", param)
	var rows *sql.Rows
	var list []listsupplier
	var city, state, country, phone sql.NullString
	sqlStatement1 := `select vendorid,vendorname,contactname,city,state,country, groupname, phone,isactive from dbo.SupplierGrid %s`
	rows, err = db.Query(fmt.Sprintf(sqlStatement1, param))
	defer rows.Close()
	for rows.Next() {
		var list1 listsupplier
		err = rows.Scan(&list1.VendorId, &list1.Name, &list1.ContactPerson, &city, &state,
			&country, &list1.Group, &phone,&list1.IsActive)
		list1.City = city.String
		list1.State = state.String
		list1.Country = country.String
		list1.Phone = phone.String
		list = append(list, list1)
	}
	res, _ := json.Marshal(list)
	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(listSuppliers)
}
